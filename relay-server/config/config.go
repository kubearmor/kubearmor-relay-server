package config

import (
	"flag"

	"github.com/spf13/viper"
)

type RelayConfig struct {
	GRPC        string
	TLSEnabled  bool
	TLSCertPath string
}

var GlobalConfig RelayConfig

const (
	ConfigGRPC       string = "gRPCPort"
	ConfigTLSEnabled string = "tlsEnabled"
	ConfigTLSCert    string = "tlsCert"
)

func readCmdLineParams() {
	grpcStr := flag.String(ConfigGRPC, "32767", "gRPC port")
	tlsEnabled := flag.Bool(ConfigTLSEnabled, true, "enble tls to connect with kubearmor ssl service")
	tlsCertPath := flag.String(ConfigTLSCert, "/var/lib/kubearmor/tls", "path to tls certs file ca.crt")

	flag.Parse()
	viper.SetDefault(ConfigGRPC, grpcStr)
	viper.SetDefault(ConfigTLSEnabled, tlsEnabled)
	viper.SetDefault(ConfigTLSCert, tlsCertPath)
}

func LoadConfig() error {

	readCmdLineParams()

	GlobalConfig.GRPC = viper.GetString(ConfigGRPC)
	GlobalConfig.TLSEnabled = viper.GetBool(ConfigTLSEnabled)
	GlobalConfig.TLSCertPath = viper.GetString(ConfigTLSCert)
	return nil
}
