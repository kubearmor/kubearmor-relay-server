module github.com/kubearmor/kubearmor-relay-server/relay-server

go 1.15

replace (
	github.com/kubearmor/kubearmor-relay-server/relay-server => ./
	github.com/kubearmor/kubearmor-relay-server/relay-server/common => ./common
	github.com/kubearmor/kubearmor-relay-server/relay-server/log => ./log
	github.com/kubearmor/kubearmor-relay-server/relay-server/server => ./server
)

require (
	github.com/google/uuid v1.3.0
	github.com/kubearmor/KubeArmor/protobuf v0.0.0-20230426155201-4a0d0af2a5d6
	go.uber.org/zap v1.19.0
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.55.0
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
)
