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

	//BroadcastPolicies bool
	//HTTPListenerPort string
	//KubeArmorPolicyServicePort string
}

var GlobalCfg KubeArmorRelayConfig

func LoadConfig() error {
	// get arguments
	gRPCPortPtr := flag.String("gRPCPort", "32767", "gRPC port to forward logs to downstream clients and receive policies on")
	k8s := flag.Bool("k8s", true, "to run relay server in k8s mode")
	//broadcastPoliciesPtr := flag.Bool("broadcastPolicies", false, "to run in policy broadcast mode")
	//kubearmorPolicyServicePortPtr := flag.String("kubearmorPolicyServicePort", "32767", "port on which kubearmor receives broadcasted policies")
	//httpListenerPortPtr := flag.String("httpPort", "2801", "http port for receiving pushed based logs from kubearmor")

	flag.Parse()

	GlobalCfg = KubeArmorRelayConfig{
		GRPCPort: *gRPCPortPtr,
		K8s:      *k8s,
		//BroadcastPolicies: *broadcastPoliciesPtr,
		//KubeArmorPolicyServicePort: *kubearmorPolicyServicePortPtr,
		//HTTPListenerPort: *httpListenerPortPtr,
	}

	kg.Printf("Final Configuration [%+v]", GlobalCfg)

	return nil
}
