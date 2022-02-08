module github.com/kubearmor/kubearmor-relay-server/relay-server

go 1.15

replace (
	github.com/kubearmor/kubearmor-relay-server/relay-server => ./
	github.com/kubearmor/kubearmor-relay-server/relay-server/common => ./common
	github.com/kubearmor/kubearmor-relay-server/relay-server/log => ./log
	github.com/kubearmor/kubearmor-relay-server/relay-server/server => ./server
)

require (
	github.com/google/uuid v1.1.2
	github.com/kubearmor/KubeArmor/protobuf v0.0.0-20220208060003-24f471a61bec
	go.uber.org/zap v1.19.0
	google.golang.org/grpc v1.35.0
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
)
