package informers

import (
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
	"k8s.io/client-go/rest"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type PodServiceInfo struct {
	Type           string
	PodName        string
	DeploymentName string
	ServiceName    string
	NamespaceName  string
}

type ClusterCache struct {
	mu         *sync.RWMutex
	ipPodCache map[string]PodServiceInfo
}

func (cc *ClusterCache) Get(IP string) (PodServiceInfo, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	value, ok := cc.ipPodCache[IP]
	return value, ok

}
func (cc *ClusterCache) Set(IP string, pi PodServiceInfo) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	switch pi.Type {
	case "POD":

		kg.Printf("Received IP %s and cached the host %s", IP, pi.PodName)
		break
	case "SERVICE":

		kg.Printf("Received IP %s and cached the host %s", IP, pi.ServiceName)
		break

	}
	cc.ipPodCache[IP] = pi

}
func (cc *ClusterCache) Delete(IP string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	delete(cc.ipPodCache, IP)

}

type Client struct {
	k8sClient      *kubernetes.Clientset
	ClusterIPCache *ClusterCache
}

func InitializeClient() *Client {
	clusterCache := &ClusterCache{
		mu: &sync.RWMutex{},

		ipPodCache: make(map[string]PodServiceInfo),
	}
	return &Client{
		k8sClient:      getK8sClient(),
		ClusterIPCache: clusterCache,
	}
}

func StartInformers(client *Client) {

	informerFactory := informers.NewSharedInformerFactory(client.k8sClient, time.Minute*10)
	kg.Printf("informerFactory created")

	podInformer := informerFactory.Core().V1().Pods().Informer()

	kg.Printf("pod informers created")

	// Set up event handlers for Pods
	podInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {

				pod := obj.(*v1.Pod)
				deploymentName := getDeploymentNamefromPod(pod)
				podInfo := PodServiceInfo{
					Type:           "POD",
					PodName:        pod.Name,
					DeploymentName: deploymentName,
					NamespaceName:  pod.Namespace,
				}

				client.ClusterIPCache.Set(pod.Status.PodIP, podInfo)

				// kg.Printf("POD Added: %s/%s, remoteIP %s\n", pod.Name, deploymentName, pod.Status.PodIP)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {

				pod := newObj.(*v1.Pod)
				deploymentName := getDeploymentNamefromPod(pod)
				podInfo := PodServiceInfo{

					Type:           "POD",
					PodName:        pod.Name,
					DeploymentName: deploymentName,

					NamespaceName: pod.Namespace,
				}

				client.ClusterIPCache.Set(pod.Status.PodIP, podInfo)
				// kg.Printf("POD Updated: %s/%s, remoteIP %s\n", pod.Name, deploymentName, pod.Status.PodIP)

			},
			DeleteFunc: func(obj interface{}) {

				pod := obj.(*v1.Pod)

				client.ClusterIPCache.Delete(pod.Status.PodIP)
			},
		},
	)

	// Get the Service informer
	serviceInformer := informerFactory.Core().V1().Services().Informer()

	// Set up event handlers
	serviceInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				service := obj.(*v1.Service)

				svcInfo := PodServiceInfo{

					Type:           "SERVICE",
					ServiceName:    service.Name,
					DeploymentName: service.Name,

					NamespaceName: service.Namespace,
				}
				client.ClusterIPCache.Set(service.Spec.ClusterIP, svcInfo)

				// kg.Printf("Service Added: %s/%s, remoteIP %s\n", service.Namespace, service.Name, service.Spec.ClusterIP)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				service := newObj.(*v1.Service)

				svcInfo := PodServiceInfo{

					Type:           "SERVICE",
					ServiceName:    service.Name,
					DeploymentName: service.Name,
					NamespaceName:  service.Namespace,
				}
				client.ClusterIPCache.Set(service.Spec.ClusterIP, svcInfo)
				kg.Printf("Service Updated: %s/%s\n", service.Namespace, service.Name)
			},
			DeleteFunc: func(obj interface{}) {
				service := obj.(*v1.Service)

				client.ClusterIPCache.Delete(service.Spec.ClusterIP)
				// kg.Printf("Service Deleted: %s/%s\n", service.Namespace, service.Name)
			},
		},
	)

	// Start the informer
	stopCh := make(chan struct{})
	defer close(stopCh)

	informerFactory.Start(stopCh)
	informerFactory.WaitForCacheSync(stopCh)

	// Wait for signals to exit
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}

func getK8sClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		kg.Errf("Error creating Kubernetes config: %v\n", err)
		return nil
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		kg.Errf("Error creating Kubernetes clientset: %v\n", err)
		return nil
	}
	kg.Printf("Successfully created k8sClientSet")
	return clientset
}

func getDeploymentNamefromPod(pod *v1.Pod) string {
	for _, ownerReference := range pod.OwnerReferences {
		switch ownerReference.Kind {
		case "ReplicaSet":
			return getDeploymentNameFromReplicaSetName(ownerReference.Name)
		case "Deployment", "DaemonSet", "StatefulSet", "Job", "CronJob", "ReplicationController":
			return ownerReference.Name
		}
	}
	return ""
}

func getDeploymentNameFromReplicaSetName(replicaSetName string) string {
	// Assuming the ReplicaSet name is in the format of `deploymentname-randomsuffix`
	// Split the name by the last dash
	parts := strings.Split(replicaSetName, "-")
	if len(parts) < 2 {
		return replicaSetName // If not in the expected format, return the original name
	}
	return strings.Join(parts[:len(parts)-1], "-")
}
