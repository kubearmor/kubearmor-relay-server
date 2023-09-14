package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	tp "github.com/kubearmor/KubeArmor/KubeArmor/types"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

// Implements PolicyService Server for recieving KubeArmor policies over gRPC
type PolicyReceiverServer struct {
}

func (pr *PolicyReceiverServer) ContainerPolicy(c context.Context, data *pb.Policy) (*pb.Response, error) {
	policyEvent := tp.K8sKubeArmorPolicyEvent{}
	res := new(pb.Response)

	err := json.Unmarshal(data.Policy, &policyEvent)

	if err == nil {
		if policyEvent.Object.Metadata.Name != "" {
			ContainerPolicyLock.Lock()
			for uid := range ContainerPolicyStructs {
				select {
				case ContainerPolicyStructs[uid].Broadcast <- data:
				default:
				}
			}
			ContainerPolicyLock.Unlock()
			res.Status = 1
		} else {
			kg.Warn("Empty Container Policy event")
			res.Status = 0
		}
	} else {
		kg.Warn("Invalid Container Policy Event")
		res.Status = 0
	}

	pol, _ := json.Marshal(data)
	fmt.Printf("%s\n", string(pol))

	kg.Printf("Pushing container policy %s", policyEvent.Object.Metadata.Name)

	if policyEvent.Type == "ADDED" || policyEvent.Type == "MODIFIED" {
		backupPolicy(policyEvent)
	} else if policyEvent.Type == "DELETED" {
		removePolicy(policyEvent.Object.Metadata.Name, containerPolicyType)
	}

	return res, nil
}

func (pr *PolicyReceiverServer) HostPolicy(c context.Context, data *pb.Policy) (*pb.Response, error) {
	policyEvent := tp.K8sKubeArmorHostPolicyEvent{}
	res := new(pb.Response)

	err := json.Unmarshal(data.Policy, &policyEvent)

	if err == nil {
		if policyEvent.Object.Metadata.Name != "" {
			HostPolicyLock.Lock()
			for uid := range HostPolicyStructs {
				select {
				case HostPolicyStructs[uid].Broadcast <- data:
				default:
				}
			}
			HostPolicyLock.Unlock()
			res.Status = 1
		} else {
			kg.Warn("Empty host policy Event")
			res.Status = 0
		}
	} else {
		kg.Warn("Invalid host policy Event")
		res.Status = 0
	}

	pol, _ := json.Marshal(data)
	fmt.Printf("%s\n", string(pol))

	kg.Printf("Pushing Host Policy %s", policyEvent.Object.Metadata.Name)

	if policyEvent.Type == "ADDED" || policyEvent.Type == "MODIFIED" {
		backupPolicy(policyEvent)
	} else if policyEvent.Type == "DELETED" {
		removePolicy(policyEvent.Object.Metadata.Name, hostPolicyType)
	}

	return res, nil
}

// backupPolicy saves policy to filesystem so that it can be broadcasted to new clients
func backupPolicy(policyEvent interface{}) {
	var policyDir, policyName string

	switch policyEvent.(type) {
	case tp.K8sKubeArmorPolicyEvent:
		policyName = policyEvent.(tp.K8sKubeArmorPolicyEvent).Object.Metadata.Name
		policyDir = containerPolicyDir
	case tp.K8sKubeArmorHostPolicyEvent:
		policyName = policyEvent.(tp.K8sKubeArmorHostPolicyEvent).Object.Metadata.Name
		policyDir = hostPolicyDir
	default:
		kg.Warnf("Failed to determine policy type")
		return
	}

	if _, err := os.Stat(policyDir); err != nil {
		if err = os.MkdirAll(policyDir, 0700); err != nil {
			kg.Warnf("Dir creation failed for dir:[%v] err:[%s]", policyDir, err.Error())
			return
		}
	}

	var file *os.File
	var err error

	if file, err = os.Create(filepath.Join(policyDir, policyName+".yaml")); err == nil {
		if policyBytes, err := json.Marshal(policyEvent); err == nil {
			if _, err = file.Write(policyBytes); err == nil {
				if err := file.Close(); err != nil {
					kg.Errf(err.Error())
				}
			}
		}
	}
}

// removePolicy removes policies from the fs
func removePolicy(name, policyType string) {
	var policyDir string

	switch policyType {
	case containerPolicyType:
		policyDir = containerPolicyDir
	case hostPolicyType:
		policyDir = hostPolicyDir
	default:
		return
	}

	fname := filepath.Join(policyDir, name+".yaml")
	if _, err := os.Stat(fname); err != nil {
		kg.Printf("Backup policy [%v] not exist", fname)
		return
	}

	if err := os.Remove(fname); err != nil {
		kg.Errf("unable to delete file:%s err=%s", fname, err.Error())
	}
}
