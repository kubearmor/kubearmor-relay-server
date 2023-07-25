package config

import (
	"flag"

	kg "github.com/kubearmor/KubeArmor/KubeArmor/log"
)

type KubeArmorRelayConfig struct {
	// common config
	GRPCPort string

	// non-k8s config
	K8s bool
}

var GlobalCfg KubeArmorRelayConfig

func LoadConfig() error {
	// get arguments
	gRPCPortPtr := flag.String("gRPCPort", "32767", "gRPC port to forward logs to downstream clients and receive policies on")
	k8s := flag.Bool("k8s", true, "to run relay server in k8s mode")

	flag.Parse()

	GlobalCfg = KubeArmorRelayConfig{
		GRPCPort: *gRPCPortPtr,
		K8s:      *k8s,
	}

	kg.Printf("Final Configuration [%+v]", GlobalCfg)

	return nil
}
