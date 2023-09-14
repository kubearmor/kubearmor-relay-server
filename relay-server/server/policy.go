package server

import (
	"path/filepath"

	"github.com/google/uuid"
	kf "github.com/kubearmor/KubeArmor/KubeArmor/feeder"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

const (
	// TODO: accept as part of configuration
	PolicyDir = "/opt/kubearmor-relay/policies/"

	containerPolicyType = "CONTAINER"
	hostPolicyType      = "HOST"
)

var (
	containerPolicyDir = filepath.Join(PolicyDir, "container")
	hostPolicyDir      = filepath.Join(PolicyDir, "host")
)

func addPolicyStruct(filter string) (string, chan *pb.Policy) {
	uid := uuid.Must(uuid.NewRandom()).String()
	conn := make(chan *pb.Policy, 1)

	policyStruct := kf.EventStruct[pb.Policy]{
		Filter:    filter,
		Broadcast: conn,
	}

	switch filter {
	case containerPolicyType:
		ContainerPolicyLock.Lock()
		ContainerPolicyStructs[uid] = policyStruct
		ContainerPolicyLock.Unlock()
	case hostPolicyType:
		HostPolicyLock.Lock()
		HostPolicyStructs[uid] = policyStruct
		HostPolicyLock.Unlock()
	}

	return uid, conn
}

func removePolicyStruct(filter, uid string) {
	switch filter {
	case containerPolicyType:
		ContainerPolicyLock.Lock()
		delete(ContainerPolicyStructs, uid)
		ContainerPolicyLock.Unlock()

		kg.Printf("Deleted the ContainerPolicy client (%s)", uid)
	case hostPolicyType:
		HostPolicyLock.Lock()
		delete(HostPolicyStructs, uid)
		HostPolicyLock.Unlock()

		kg.Printf("Deleted the HostPolicy client (%s)", uid)
	}

	return
}
