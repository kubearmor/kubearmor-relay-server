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
	github.com/google/uuid v1.4.0
	github.com/kubearmor/KubeArmor/protobuf v0.0.0-20240208122905-ace82ae0506c
	go.uber.org/zap v1.19.0
	google.golang.org/grpc v1.61.0
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
)
