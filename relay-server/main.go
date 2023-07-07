// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of KubeArmor

package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
	"github.com/kubearmor/kubearmor-relay-server/relay-server/server"
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
	relayServer := server.NewRelayServer(*gRPCPortPtr)
	if relayServer == nil {
		kg.Warnf("Failed to create a relay server (:%s)", *gRPCPortPtr)
		return
	}
	kg.Printf("Created a relay server (:%s)", *gRPCPortPtr)

	// serve log feeds (to clients)
	go relayServer.ServeLogFeeds()
	kg.Print("Started to serve gRPC-based log feeds")

	// // get log feeds (from KubeArmor)
	// go relayServer.GetFeedsFromNodes()
	// kg.Print("Started to receive log feeds from each node")

	// get log feeds (from BlueLock)
	go relayServer.ListenOnHTTP()
	kg.Print("Started to receive log feeds from BlueLock")

	// listen for interrupt signals
	sigChan := GetOSSigChannel()
	<-sigChan
	close(StopChan)

	// destroy the relay server
	if err := relayServer.DestroyRelayServer(); err != nil {
		kg.Warnf("Failed to destroy the relay server (%s)", err.Error())
		return
	}
	kg.Print("Destroyed the relay server")

	// == //
}
