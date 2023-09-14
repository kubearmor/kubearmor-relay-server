package config

import (
	"flag"

	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

type KubeArmorRelayConfig struct {
	// common config

	// port which clients of relay server connect to
	GRPCPort string

	// to enable reverse logger
	EnableReverseLogClient bool

	// to enable policy server
	EnablePolicyServer bool
}

var GlobalCfg KubeArmorRelayConfig

func LoadConfig() error {
	// get arguments
	gRPCPortPtr := flag.String("gRPCPort", "32767", "gRPC port to forward logs to downstream clients and receive policies on")
	enableReverseLogClient := flag.Bool("enableReverseLogClient", false, "to receive logs from KubeArmor's reverse logger")
	enablePolicyServer := flag.Bool("enablePolicyServer", false, "to receive logs from KubeArmor's reverse logger")

	flag.Parse()

	GlobalCfg = KubeArmorRelayConfig{
		GRPCPort:               *gRPCPortPtr,
		EnableReverseLogClient: *enableReverseLogClient,
		EnablePolicyServer:     *enablePolicyServer,
	}

	kg.Printf("Final Configuration [%+v]", GlobalCfg)

	return nil
}
