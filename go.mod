module github.com/kubearmor/kubearmor-relay-server

go 1.15

replace (
	github.com/kubearmor/kubearmor-relay-server => ./
	github.com/kubearmor/kubearmor-relay-server/core => ./core
	github.com/kubearmor/kubearmor-relay-server/log => ./log
)

require (
	github.com/google/uuid v1.1.2
	github.com/kubearmor/KubeArmor/protobuf v0.0.0-20210706103022-a88ee52bbf8a
	go.uber.org/zap v1.19.0
	google.golang.org/grpc v1.35.0
	k8s.io/api v0.21.1
	k8s.io/client-go v0.21.1
)
