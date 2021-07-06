module github.com/kubearmor/kubearmor-relay-server

go 1.15

replace (
	github.com/kubearmor/kubearmor-relay-server => ./
	github.com/kubearmor/kubearmor-relay-server/core => ./core
)

require (
	github.com/kubearmor/KubeArmor/protobuf v0.0.0-20210706103022-a88ee52bbf8a // indirect
	github.com/kubearmor/kubearmor-relay-server/core v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.35.0 // indirect
)
