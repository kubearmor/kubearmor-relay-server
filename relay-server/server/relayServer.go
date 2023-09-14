// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of KubeArmor

package server

import (
	"net"
	"sync"
	"time"

	kf "github.com/kubearmor/KubeArmor/KubeArmor/feeder"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	cfg "github.com/kubearmor/kubearmor-relay-server/relay-server/config"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// ============ //
// == Global == //
// ============ //

var (
	// Running flag
	Running bool

	// ClientList Map
	ClientList map[string]int

	// ClientListLock Lock
	ClientListLock *sync.Mutex

	// For policy managemnt
	ContainerPolicyStructs map[string]kf.EventStruct[pb.Policy]
	ContainerPolicyLock    sync.RWMutex

	HostPolicyStructs map[string]kf.EventStruct[pb.Policy]
	HostPolicyLock    sync.RWMutex
)

func init() {
	Running = true
	ClientList = map[string]int{}
	ClientListLock = &sync.Mutex{}
}

// ================== //
// == Relay Server == //
// ================== //

// RelayServer Structure
type RelayServer struct {
	// Generic //

	// port
	Port string

	// gRPC listener
	Listener net.Listener

	// grpc server
	GRPCServer *grpc.Server

	// wait group
	WgServer sync.WaitGroup

	// Log Relay //

	EventStructs kf.EventStructs
}

// NewRelayServer Function
func NewRelayServer() *RelayServer {
	rs := &RelayServer{}

	rs.Port = cfg.GlobalCfg.GRPCPort

	// listen to gRPC port
	listener, err := net.Listen("tcp", ":"+rs.Port)
	if err != nil {
		kg.Errf("Failed to listen a port (%s)\n", rs.Port)
		return nil
	}
	rs.Listener = listener

	kaep := keepalive.EnforcementPolicy{
		PermitWithoutStream: true,
	}

	kasp := keepalive.ServerParameters{
		Time:    1 * time.Second,
		Timeout: 5 * time.Second,
	}

	// create a grpc server
	rs.GRPCServer = grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))

	// initialize event structs
	rs.EventStructs = kf.EventStructs{
		MsgStructs: make(map[string]kf.EventStruct[pb.Message]),
		MsgLock:    sync.RWMutex{},

		AlertStructs: make(map[string]kf.EventStruct[pb.Alert]),
		AlertLock:    sync.RWMutex{},

		LogStructs: make(map[string]kf.EventStruct[pb.Log]),
		LogLock:    sync.RWMutex{},
	}

	ContainerPolicyStructs = make(map[string]kf.EventStruct[pb.Policy])
	ContainerPolicyLock = sync.RWMutex{}

	HostPolicyStructs = make(map[string]kf.EventStruct[pb.Policy])
	HostPolicyLock = sync.RWMutex{}

	// health server
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(rs.GRPCServer, healthServer)

	// register a log service
	logService := &LogService{
		EventStructs: &rs.EventStructs,
	}
	pb.RegisterLogServiceServer(rs.GRPCServer, logService)

	// register ReverseLogService server
	if cfg.GlobalCfg.EnableReverseLogClient {
		pushLogService := &ReverseLogService{
			EventStructs: &rs.EventStructs,
		}
		pb.RegisterReverseLogServiceServer(rs.GRPCServer, pushLogService)
		healthServer.SetServingStatus(pb.ReverseLogService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)
	}

	if cfg.GlobalCfg.EnablePolicyServer {
		// enable the general policy server (receiver) by default
		policyService := &PolicyReceiverServer{}
		pb.RegisterPolicyServiceServer(rs.GRPCServer, policyService)
		healthServer.SetServingStatus(pb.PolicyService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)
		kg.Printf("Registered PolicyServer for receiving policies on port %s", rs.Port)

		// reverse policy server
		reversePolicyService := &ReversePolicyServer{}
		pb.RegisterReversePolicyServiceServer(rs.GRPCServer, reversePolicyService)
		healthServer.SetServingStatus(pb.ReversePolicyService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)
		kg.Printf("Registered ReversePolicyServer for sending policies to KubeArmor")
	}

	reflection.Register(rs.GRPCServer)

	// set wait group
	rs.WgServer = sync.WaitGroup{}

	return rs
}

// DestroyRelayServer Function
func (rs *RelayServer) DestroyRelayServer() error {
	// stop gRPC service
	Running = false

	// wait for a while
	time.Sleep(time.Second * 1)

	// close listener
	if rs.Listener != nil {
		if err := rs.Listener.Close(); err != nil {
			kg.Err(err.Error())
		}
		rs.Listener = nil
	}

	// wait for other routines
	rs.WgServer.Wait()

	return nil
}

// =============== //
// == Log Feeds == //
// =============== //

// ServeLogFeeds Function
func (rs *RelayServer) ServeLogFeeds() {
	rs.WgServer.Add(1)
	defer rs.WgServer.Done()

	// feed logs
	if err := rs.GRPCServer.Serve(rs.Listener); err != nil {
		kg.Print("Terminated the gRPC service")
	}
}

// Remove nodeIP from ClientList
func DeleteClientEntry(nodeIP string) {
	ClientListLock.Lock()
	defer ClientListLock.Unlock()

	_, exists := ClientList[nodeIP]

	if exists {
		delete(ClientList, nodeIP)
	}
}

// =============== //
// == KubeArmor == //
// =============== //

func connectToKubeArmorK8s(nodeIP, port string) error {

	// create connection info
	server := nodeIP + ":" + port

	defer DeleteClientEntry(nodeIP)

	// create a client
	client := NewClient(server)
	if client == nil {
		return nil
	}

	// do healthcheck
	if ok := client.DoHealthCheck(); !ok {
		kg.Warnf("Failed to check the liveness of KubeArmor's gRPC service (%s)", server)
		return nil
	}
	kg.Printf("Checked the liveness of KubeArmor's gRPC service (%s)", server)

	client.WgServer.Add(1)
	// watch messages
	go client.WatchMessages()
	kg.Print("Started to watch messages from " + server)

	client.WgServer.Add(1)
	// watch alerts
	go client.WatchAlerts()
	kg.Print("Started to watch alerts from " + server)

	client.WgServer.Add(1)
	// watch logs
	go client.WatchLogs()
	kg.Print("Started to watch logs from " + server)

	time.Sleep(time.Second * 1)
	// wait for other routines
	client.WgServer.Wait()

	if err := client.DestroyClient(); err != nil {
		kg.Warnf("Failed to destroy the client (%s)", server)
	}
	kg.Printf("Destroyed the client (%s)", server)

	return nil
}

// GetFeedsFromNodes Function
func (rs *RelayServer) GetFeedsFromK8sNodes() {

	rs.WgServer.Add(1)
	defer rs.WgServer.Done()

	kg.Print("Initialized the Kubernetes client")

	for Running {
		newNodes := K8s.GetKubeArmorNodes()

		for _, nodeIP := range newNodes {
			ClientListLock.Lock()
			if _, ok := ClientList[nodeIP]; !ok {
				ClientList[nodeIP] = 1
				go connectToKubeArmorK8s(nodeIP, rs.Port)
			}
			ClientListLock.Unlock()
		}

		time.Sleep(time.Second * 1)
	}
}
