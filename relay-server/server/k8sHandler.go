// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of KubeArmor

// Package server exports kubearmor logs
package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	kl "github.com/kubearmor/kubearmor-relay-server/relay-server/common"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ================= //
// == K8s Handler == //
// ================= //

// K8s Handler
var K8s *K8sHandler

// init Function
func init() {
	K8s = NewK8sHandler()
}

// K8sHandler Structure
type K8sHandler struct {
	K8sClient   *kubernetes.Clientset
	HTTPClient  *http.Client
	WatchClient *http.Client

	K8sToken string
	K8sHost  string
	K8sPort  string
}

var stdoutlogs = false
var stdoutalerts = false
var stdoutmsg = false

// NewK8sHandler Function
func NewK8sHandler() *K8sHandler {
	kh := &K8sHandler{}

	if val, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		kh.K8sHost = val
	} else {
		kh.K8sHost = "127.0.0.1"
	}

	if val, ok := os.LookupEnv("KUBERNETES_PORT_443_TCP_PORT"); ok {
		kh.K8sPort = val
	} else {
		kh.K8sPort = "8001" // kube-proxy
	}

	//Enable printing logs
	if val, ok := os.LookupEnv("ENABLE_STDOUT_LOGS"); ok {
		ValueLower := strings.ToLower(val)
		if ValueLower == "true" {
			stdoutlogs = true
		}
	}
	//Enable printing Alerts
	if val, ok := os.LookupEnv("ENABLE_STDOUT_ALERTS"); ok {
		ValueLower := strings.ToLower(val)
		if ValueLower == "true" {
			stdoutalerts = true
		}
	}
	//Enable printing MSgs
	if val, ok := os.LookupEnv("ENABLE_STDOUT_MSGS"); ok {
		ValueLower := strings.ToLower(val)
		if ValueLower == "true" {
			stdoutmsg = true
		}
	}

	kh.HTTPClient = &http.Client{
		Timeout: time.Second * 5,
		// #nosec
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	kh.WatchClient = &http.Client{
		// #nosec
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return kh
}

// ================ //
// == K8s Client == //
// ================ //

// InitK8sClient Function
func (kh *K8sHandler) InitK8sClient() bool {
	if !kl.IsK8sEnv() { // not Kubernetes
		return false
	}

	if kh.K8sClient == nil {
		if kl.IsInK8sCluster() {
			return kh.InitInclusterAPIClient()
		}
		return kh.InitLocalAPIClient()
	}

	return true
}

// InitLocalAPIClient Function
func (kh *K8sHandler) InitLocalAPIClient() bool {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
		if _, err := os.Stat(filepath.Clean(kubeconfig)); err != nil {
			return false
		}
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return false
	}

	// creates the clientset
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return false
	}
	kh.K8sClient = client

	return true
}

// InitInclusterAPIClient Function
func (kh *K8sHandler) InitInclusterAPIClient() bool {
	read, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return false
	}
	kh.K8sToken = string(read)

	// create the configuration by token
	kubeConfig := &rest.Config{
		Host:        "https://" + kh.K8sHost + ":" + kh.K8sPort,
		BearerToken: kh.K8sToken,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return false
	}
	kh.K8sClient = client

	return true
}

// ============== //
// == API Call == //
// ============== //

// DoRequest Function
func (kh *K8sHandler) DoRequest(cmd string, data interface{}, path string) ([]byte, error) {
	URL := ""

	if kl.IsInK8sCluster() {
		URL = "https://" + kh.K8sHost + ":" + kh.K8sPort
	} else {
		URL = "http://" + kh.K8sHost + ":" + kh.K8sPort
	}

	pbytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(cmd, URL+path, bytes.NewBuffer(pbytes))
	if err != nil {
		return nil, err
	}

	if kl.IsInK8sCluster() {
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", kh.K8sToken))
	}

	resp, err := kh.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := resp.Body.Close(); err != nil {
		kg.Err(err.Error())
	}

	return resBody, nil
}

func (kh *K8sHandler) WatchKubeArmorPods(ctx context.Context, wg *sync.WaitGroup, ipsChan chan string) {
	defer func() {
		close(ipsChan)
		wg.Done()
	}()

	// Get the KubeArmor pods IP that were added before relay itself.
	once := sync.Once{}
	once.Do(func() {
		kh.findExistingKaPodsIp(ctx, ipsChan)
	})

	podInformer := kh.getKaPodInformer(ipsChan)
	podInformer.Run(ctx.Done())
}

func (kh *K8sHandler) getKaPodInformer(ipsChan chan string) cache.SharedIndexInformer {
	option := informers.WithTweakListOptions(func(lo *metav1.ListOptions) {
		lo.LabelSelector = "kubearmor-app=kubearmor"
	})

	factory := informers.NewSharedInformerFactoryWithOptions(kh.K8sClient, 0, option)
	informer := factory.Core().V1().Pods().Informer()

	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if ok {
				if pod.Status.PodIP != "" {
					ipsChan <- pod.Status.PodIP
				}
			}
		},
		UpdateFunc: func(old, new interface{}) {
			newPod, ok := new.(*corev1.Pod)
			if ok {
				if newPod.Status.PodIP != "" {
					ipsChan <- newPod.Status.PodIP
				}
			}
		},
	})

	return informer
}

func (kh *K8sHandler) findExistingKaPodsIp(ctx context.Context, ipsChan chan string) {
	pods, err := kh.K8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "kubearmor-app=kubearmor",
	})

	if err != nil {
		kg.Errf("failed to list KubeArmor pods: %v", err)
		return
	}

	for _, pod := range pods.Items {
		if pod.Status.PodIP != "" {
			ipsChan <- pod.Status.PodIP
		}
	}
}
