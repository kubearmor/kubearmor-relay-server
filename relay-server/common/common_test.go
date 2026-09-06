// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of KubeArmor

package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsK8sLocalWithoutKubeconfig(t *testing.T) {
	t.Setenv("KUBECONFIG", "")
	t.Setenv("HOME", t.TempDir())

	if IsK8sLocal() {
		t.Error("IsK8sLocal() = true, want false when neither KUBECONFIG nor ~/.kube/config exist")
	}
}

func TestIsK8sLocalWithKubeconfig(t *testing.T) {
	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "config")
	if err := os.WriteFile(kubeconfig, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", kubeconfig)

	if !IsK8sLocal() {
		t.Error("IsK8sLocal() = false, want true when KUBECONFIG points to an existing file")
	}
}
