// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of KubeArmor

package common

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ============ //
// == Common == //
// ============ //

// Clone Function
func Clone(src, dst interface{}) error {
	arr, _ := json.Marshal(src)
	return json.Unmarshal(arr, dst)
}

// ================ //
// == Kubernetes == //
// ================ //

// IsK8sLocal Function
func IsK8sLocal() bool {
	k8sConfig := os.Getenv("KUBECONFIG")
	if _, err := os.Stat(filepath.Clean(k8sConfig)); err == nil {
		return true
	}

	home := os.Getenv("HOME")
	if _, err := os.Stat(filepath.Clean(home + "/.kube/config")); err == nil {
		return true
	}

	return false
}

// IsInK8sCluster Function
func IsInK8sCluster() bool {
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		return true
	}

	if _, err := os.Stat(filepath.Clean("/run/secrets/kubernetes.io")); err == nil {
		return true
	}

	return false
}

// IsK8sEnv Function
func IsK8sEnv() bool {
	// local
	if IsK8sLocal() {
		return true
	}

	// in-cluster
	if IsInK8sCluster() {
		return true
	}

	return false
}
