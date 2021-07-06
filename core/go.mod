module github.com/kubearmor/kubearmor-relay-server/core

go 1.15

replace (
	github.com/kubearmor/kubearmor-relay-server => ../
	github.com/kubearmor/kubearmor-relay-server/core => ./
)

require (
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
)
