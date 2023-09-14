// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of KubeArmor

package main

import (
	"os"
	"os/signal"
	"syscall"

	cfg "github.com/kubearmor/kubearmor-relay-server/relay-server/config"
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
	err := cfg.LoadConfig()
	if err != nil {
		kg.Errf("Failed to load config %s", err)
	}

	// create a relay server
	relayServer := server.NewRelayServer()
	if relayServer == nil {
		kg.Warnf("Failed to create a relay server (:%s)", cfg.GlobalCfg.GRPCPort)
		return
	}
	kg.Printf("Created a relay server (:%s)", cfg.GlobalCfg.GRPCPort)

	// serve log feeds (to clients)
	go relayServer.ServeLogFeeds()
	kg.Print("Started to serve gRPC-based log feeds")

	// get log feeds (from K8s nodes running KubeArmor)
	if server.K8s.InitK8sClient() {
		go relayServer.GetFeedsFromK8sNodes()
		kg.Print("Started to receive log feeds from K8s nodes")
	}

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
