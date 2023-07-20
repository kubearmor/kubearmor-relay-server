package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	tp "github.com/kubearmor/KubeArmor/KubeArmor/types"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	PolicyDir = "/opt/kubearmor-relay/policies"

	PolicyStructs map[string]PolicyStruct
	PolicyLock *sync.RWMutex
)

type PolicyStruct struct {
	Broadcast chan *pb.Policy
}

type PolicyStreamerServer struct {
	//
}

func (ps *PolicyStreamerServer) Init() {
	// initialize Policy structs
	PolicyStructs = make(map[string]PolicyStruct)
	PolicyLock = &sync.RWMutex{}
}

func (ps *PolicyStreamerServer) HealthCheck(ctx context.Context, nonce *pb.HealthCheckReq) (*pb.HealthCheckReply, error) {
	replyMessage := pb.HealthCheckReply{Retval: nonce.Nonce}
	return &replyMessage, nil
}

func (ps *PolicyStreamerServer) ContainerPolicy(rs *pb.Response, svr pb.PolicyStreamService_ContainerPolicyServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()
	conn := make(chan *pb.Policy, 1)
	defer close(conn)

	policyStruct := PolicyStruct{}
	policyStruct.Broadcast = conn
	PolicyLock.Lock()
	PolicyStructs[uid] = policyStruct
	PolicyLock.Unlock()

	kg.Printf("Added a new client (%s) for ContainerPolicy", uid)

	defer removePolicyStruct(uid)

	// send all policies to this client
	policyFiles := make([]fs.DirEntry, 1)
	if _, err := os.Stat(PolicyDir); err != nil {
		kg.Warnf("Policies dir not found for broadcast: %s", err)
	} else {
		if policyFiles, err = os.ReadDir(PolicyDir); err == nil {
			if len(policyFiles) == 0 {
				kg.Warn("No policies found for broadcasting")
			}

			for _, file := range policyFiles {
				if data, err := os.ReadFile(filepath.Join(PolicyDir, file.Name())); err == nil {
					resp := &pb.Policy{
						Policy: data,
					}

					kg.Printf("Pushing policy %s to new client %s", file.Name(), uid)
					if status, ok := status.FromError(svr.Send(resp)); ok {
						switch status.Code() {
						case codes.OK:
							// noop
						case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
							kg.Warnf("relay failed to send a policy=[%+v] err=[%s]", resp, status.Err().Error())
						}
					}
				}
			}
		} else {
			kg.Errf("Failed to find local policies. %s", err)
		}
	}

	for Running {
		select {
		case <-svr.Context().Done():
			return nil
		case resp := <-conn:
			if status, ok := status.FromError(svr.Send(resp)); ok {
				switch status.Code() {
				case codes.OK:
					// noop
				case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
					kg.Warnf("relay failed to send a policy=[%+v] err=[%s]", resp, status.Err().Error())
				default:
					return nil
				}
			}
		}
	}

	return nil
}

// removePolicyStruct Function
func removePolicyStruct (uid string) {
	PolicyLock.Lock()
	defer PolicyLock.Unlock()

	delete(PolicyStructs, uid)

	kg.Printf("Deleted the client (%s) for PushContainerPolicy", uid)
}

func (ps *PolicyStreamerServer) HostPolicy(rs *pb.Response, svr pb.PolicyStreamService_HostPolicyServer) error {
	return nil
}

type PolicyReceiverServer struct {
	//pb.PolicyServiceServer
	//UpdateContainerPolicy func(tp.K8sKubeArmorPolicyEvent)
	//UpdateHostPolicy      func(tp.K8sKubeArmorHostPolicyEvent)
}

/*
func (rs *RelayServer) InitPolicyBroadcaster() {
	policyService := &policy.ServiceServer{}
	policyService.UpdateContainerPolicy = WatchPolicies
	pb.RegisterPolicyServiceServer(rs.LogServer, policyService)
}
*/

func (pr *PolicyReceiverServer) ContainerPolicy(c context.Context, data *pb.Policy) (*pb.Response, error) {
	policyEvent := tp.K8sKubeArmorPolicyEvent{}
	res := new(pb.Response)

	err := json.Unmarshal(data.Policy, &policyEvent)

	if err == nil {
		if policyEvent.Object.Metadata.Name != "" {
			PolicyLock.Lock()
			for uid := range PolicyStructs {
				select {
				case PolicyStructs[uid].Broadcast <- data:
				default:
				}
			}
			PolicyLock.Unlock()
			res.Status = 1
		} else {
			kg.Warn("Empty container policy event")
			res.Status = 0
		}
	} else {
		kg.Warn("Invalid Container Policy Event")
		res.Status = 0
	}

	pol, _ := json.Marshal(data)
	fmt.Printf("%s\n", string(pol))

	kg.Printf("Pushing policy %s", policyEvent.Object.Metadata.Name)

	//for addr := range ClientList {
	//	kg.Printf("Pushing policy %s to client %s", event.Object.Metadata.Name, addr)
	//	PushPolicy(addr, config.GlobalCfg.KubeArmorPolicyServicePort, &req)
	//}

	if policyEvent.Type == "ADDED" || policyEvent.Type == "MODIFIED" {
		backupPolicy(policyEvent)
	} else if policyEvent.Type == "DELETED" {
		removePolicy(policyEvent.Object.Metadata.Name)
	}

	return res, nil
}

func (pr *PolicyReceiverServer) HostPolicy(c context.Context, data *pb.Policy) (*pb.Response, error) {
	return nil, nil
}

/*
func PushPolicy(addr, port string, req *pb.Policy) {
	grpcURL := fmt.Sprintf("%s:%s", addr, port)
	conn, err := grpc.Dial(grpcURL, grpc.WithInsecure())
	if err != nil {
		kg.Warnf("Failed to connect with kubearmor at %s", addr)
		// it is possible that this instance is dead
		DeleteClientEntry(addr)
		return
	}

	client := pb.NewPolicyServiceClient(conn)

	resp, err := client.ContainerPolicy(context.Background(), req)
	if err != nil || resp.Status != 1 {
		kg.Warnf("Failed to send policy to kubearmor at %s", addr)
		return
	}
	kg.Printf("Pushed policy to %s successfully", addr)
}
*/


// backupPolicy saves policy to file system so that it can be broadcasted to new clients
func backupPolicy(policyEvent tp.K8sKubeArmorPolicyEvent) {
	if _, err := os.Stat(PolicyDir); err != nil {
		if err = os.MkdirAll(PolicyDir, 0700); err != nil {
			kg.Warnf("Dir creation failed for [%v]", PolicyDir)
			return
		}
	}

	var file *os.File
	var err error

	if file, err = os.Create(filepath.Join(PolicyDir, policyEvent.Object.Metadata.Name + ".yaml")); err == nil {
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
func removePolicy(name string) {
	fname := filepath.Join(PolicyDir, name + ".yaml")
	if _, err := os.Stat(fname); err != nil {
		kg.Printf("Backup policy [%v] not exist", fname)
		return
	}

	if err := os.Remove(fname); err != nil {
		kg.Errf("unable to delete file:%s err=%s", fname, err.Error())
	}
}
