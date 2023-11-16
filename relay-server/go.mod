module github.com/kubearmor/kubearmor-relay-server/relay-server

go 1.15

replace (
	github.com/kubearmor/kubearmor-relay-server/relay-server => ./
	github.com/kubearmor/kubearmor-relay-server/relay-server/common => ./common
	github.com/kubearmor/kubearmor-relay-server/relay-server/log => ./log
	github.com/kubearmor/kubearmor-relay-server/relay-server/server => ./server
)

require (
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/dustin/go-humanize v1.0.1
	github.com/elastic/go-elasticsearch/v7 v7.17.7
	github.com/google/uuid v1.3.1
	github.com/kubearmor/KubeArmor/protobuf v0.0.0-20230426155201-4a0d0af2a5d6
	github.com/spf13/viper v1.17.0
	go.uber.org/zap v1.21.0
	google.golang.org/grpc v1.58.2
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
)
