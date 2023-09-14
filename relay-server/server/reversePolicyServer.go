package server

import (
	"io/fs"
	"os"
	"path/filepath"

	kc "github.com/kubearmor/KubeArmor/KubeArmor/common"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

// implements the ReversePolicyServer which sends policies on a connection
// initiated by KubeArmor
type ReversePolicyServer struct {
	//
}

func (ps *ReversePolicyServer) ContainerPolicy(svr pb.ReversePolicyService_ContainerPolicyServer) error {
	uid, conn := addPolicyStruct(containerPolicyType)
	defer close(conn)
	defer removePolicyStruct(containerPolicyType, uid)

	kg.Printf("Added a new client (%s) for ContainerPolicy", uid)

	go ps.sendBackupPolicies(conn, containerPolicyType)

	for Running {
		select {
		case <-svr.Context().Done():
			return nil
		case resp := <-conn:
			err := svr.Send(resp)
			if grpcErr := kc.HandleGRPCErrors(err); grpcErr != nil {
				kg.Warnf("Relay failed to send a ContainerPolicy=[%+v] err=[%s]", resp, grpcErr.Error())
				return err
			}
		}
	}

	return nil
}

func (ps *ReversePolicyServer) HostPolicy(svr pb.ReversePolicyService_HostPolicyServer) error {
	uid, conn := addPolicyStruct(hostPolicyType)
	defer close(conn)
	defer removePolicyStruct(hostPolicyType, uid)

	kg.Printf("Added a new client (%s) for HostPolicy", uid)

	go ps.sendBackupPolicies(conn, hostPolicyType)

	for Running {
		select {
		case <-svr.Context().Done():
			return nil
		case resp := <-conn:
			err := svr.Send(resp)
			if grpcErr := kc.HandleGRPCErrors(err); grpcErr != nil {
				kg.Warnf("relay failed to send a host policy=[%+v] err=[%s]", resp, grpcErr.Error())
				return err
			}
		}
	}

	return nil
}

// send all policies to this new client
func (ps *ReversePolicyServer) sendBackupPolicies(conn chan *pb.Policy, policyType string) {
	var (
		policyDir string
	)

	switch policyType {
	case containerPolicyType:
		policyDir = containerPolicyDir
	case hostPolicyType:
		policyDir = hostPolicyDir
	}

	policyFiles := make([]fs.DirEntry, 1)
	if _, err := os.Stat(policyDir); err != nil {
		kg.Warnf("Policy dir %s not found for broadcast: %s", policyDir, err)
	} else {
		if policyFiles, err = os.ReadDir(policyDir); err == nil {
			if len(policyFiles) == 0 {
				kg.Debugf("No policies found for broadcasting in %s", policyDir)
			}

			for _, file := range policyFiles {
				if data, err := os.ReadFile(filepath.Join(policyDir, file.Name())); err == nil {
					resp := &pb.Policy{
						Policy: data,
					}

					kg.Printf("Pushing backup policy %s to new client", file.Name())

					select {
					// will be blocked until one policy is received by the calling function
					case conn <- resp:
					default:
					}
				}
			}
		} else {
			kg.Errf("Failed to find local policies. %s", err)
		}
	}
}
