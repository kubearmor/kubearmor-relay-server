package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/kubearmor/kubearmor-relay-server/relay-server/core"
	"github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

// StopChan Channel
var StopChan chan struct{}

// init Function
func init() {
	StopChan = make(chan struct{})
}

// ==================== //
// == Signal Handler == //
// ==================== //

// GetOSSigChannel Function
func GetOSSigChannel() chan os.Signal {
	c := make(chan os.Signal, 1)

	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		os.Interrupt)

	return c
}

// ========== //
// == Main == //
// ========== //

func main() {
	// == //

	// get arguments
	gRPCPortPtr := flag.String("gRPCPort", "32767", "gRPC port")
	flag.Parse()

	// == //

	// create a relay server
	relayServer := core.NewRelayServer(*gRPCPortPtr)
	if relayServer == nil {
		log.Warnf("Failed to create a relay server (:%s)", *gRPCPortPtr)
		return
	}
	log.Printf("Created a relay server (:%s)", *gRPCPortPtr)

	// serve log feeds
	go relayServer.ServeLogFeeds()
	log.Print("Started to serve gRPC-based log feeds")

	// get log feeds
	go relayServer.GetFeedsFromNodes()
	log.Print("Started to receive log feeds from each node")

	// listen for interrupt signals
	sigChan := GetOSSigChannel()
	<-sigChan
	close(StopChan)

	// destroy the relay server
	if err := relayServer.DestroyRelayServer(); err != nil {
		log.Warnf("Failed to destroy the relay server (%s)", err.Error())
		return
	}
	log.Print("Destroyed the relay server")

	// == //
}
