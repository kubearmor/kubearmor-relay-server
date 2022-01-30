package server

import (
	"context"
	"math/rand"
	"time"

	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"

	pb "github.com/kubearmor/KubeArmor/protobuf"
	"google.golang.org/grpc"
)

// ClientList Map
var ClientList map[string]*LogClient

func init() {
	ClientList = map[string]*LogClient{}
}

// =============== //
// == Log Feeds == //
// =============== //

// LogClient Structure
type LogClient struct {
	// flag
	Running bool

	// server
	server string

	// connection
	conn *grpc.ClientConn

	// client
	client pb.LogServiceClient

	// messages
	msgStream pb.LogService_WatchMessagesClient

	// alerts
	alertStream pb.LogService_WatchAlertsClient

	// logs
	logStream pb.LogService_WatchLogsClient
}

// NewClient Function
func NewClient(server string) *LogClient {
	lc := &LogClient{}

	lc.Running = true

	// == //

	lc.server = server

	conn, err := grpc.Dial(lc.server, grpc.WithInsecure())
	if err != nil {
		kg.Warnf("Failed to connect to KubeArmor's gRPC service (%s)\n", server)
		return nil
	}
	lc.conn = conn

	lc.client = pb.NewLogServiceClient(lc.conn)

	// == //

	msgIn := pb.RequestMessage{}
	msgIn.Filter = ""

	msgStream, err := lc.client.WatchMessages(context.Background(), &msgIn)
	if err != nil {
		kg.Warnf("Failed to call WatchMessages (%s)\n", server)
		return nil
	}
	lc.msgStream = msgStream

	// == //

	alertIn := pb.RequestMessage{}
	alertIn.Filter = "policy"

	alertStream, err := lc.client.WatchAlerts(context.Background(), &alertIn)
	if err != nil {
		kg.Warnf("Failed to call WatchAlerts (%s)\n", server)
		return nil
	}
	lc.alertStream = alertStream

	// == //

	logIn := pb.RequestMessage{}
	logIn.Filter = "system"

	logStream, err := lc.client.WatchLogs(context.Background(), &logIn)
	if err != nil {
		kg.Warnf("Failed to call WatchLogs (%s)\n", server)
		return nil
	}
	lc.logStream = logStream

	// == //

	return lc
}

// DoHealthCheck Function
func (lc *LogClient) DoHealthCheck() bool {
	// #nosec
	randNum := rand.Int31()

	// send a nonce
	nonce := pb.NonceMessage{Nonce: randNum}
	res, err := lc.client.HealthCheck(context.Background(), &nonce)
	if err != nil {
		kg.Warnf("Failed to check the liveness of KubeArmor's gRPC service (%s)", lc.server)
		return false
	}

	// check nonce
	if randNum != res.Retval {
		return false
	}

	return true
}

// WatchMessages Function
func (lc *LogClient) WatchMessages() error {
	var err error

	for lc.Running {
		var msg *pb.Message

		if msg, err = lc.msgStream.Recv(); err != nil {
			kg.Warnf("Failed to receive a message (%s)", lc.server)
			break
		}

		if msg != nil {
			MsgLock.Lock()
			defer MsgLock.Unlock()

			for uid := range MsgStructs {
				MsgStructs[uid].Queue.Push(*msg)
			}
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	lc.logStream = nil
	kg.Print("Stopped watching messages from " + lc.server)

	if err := lc.DestroyClient(); err != nil {
		kg.Err(err.Error())
	}

	return nil
}

// WatchAlerts Function
func (lc *LogClient) WatchAlerts() error {
	var err error

	for lc.Running {
		var alert *pb.Alert

		if alert, err = lc.alertStream.Recv(); err != nil {
			kg.Warnf("Failed to receive an alert (%s)", lc.server)
			break
		}

		if alert != nil {
			AlertLock.Lock()
			defer AlertLock.Unlock()

			for uid := range AlertStructs {
				AlertStructs[uid].Queue.Push(*alert)
			}
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	lc.logStream = nil
	kg.Print("Stopped watching alerts from " + lc.server)

	if err := lc.DestroyClient(); err != nil {
		kg.Err(err.Error())
	}

	return nil
}

// WatchLogs Function
func (lc *LogClient) WatchLogs() error {
	var err error

	for lc.Running {
		var log *pb.Log

		if log, err = lc.logStream.Recv(); err != nil {
			kg.Warnf("Failed to receive a log (%s)", lc.server)
			break
		}

		if log != nil {
			LogLock.Lock()
			defer LogLock.Unlock()

			for uid := range LogStructs {
				LogStructs[uid].Queue.Push(*log)
			}
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	lc.logStream = nil
	kg.Print("Stopped watching logs from " + lc.server)

	if err := lc.DestroyClient(); err != nil {
		kg.Err(err.Error())
	}

	return nil
}

// DestroyClient Function
func (lc *LogClient) DestroyClient() error {
	lc.Running = false

	if lc.msgStream == nil && lc.alertStream == nil && lc.logStream == nil {
		if err := lc.conn.Close(); err != nil {
			return err
		}
	}

	return nil
}

// =============== //
// == KubeArmor == //
// =============== //

func connectToKubeArmor(server string) *LogClient {
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

	// watch messages
	go client.WatchMessages()
	kg.Print("Started to watch messages from " + server)

	// watch alerts
	go client.WatchAlerts()
	kg.Print("Started to watch alerts from " + server)

	// watch logs
	go client.WatchLogs()
	kg.Print("Started to watch logs from " + server)

	return client
}

// GetFeedsFromNodes Function
func (rs *RelayServer) GetFeedsFromNodes() {
	rs.WgServer.Add(1)
	defer rs.WgServer.Done()

	if K8s.InitK8sClient() {
		kg.Print("Initialized the Kubernetes client")

		for Running {
			newNodes := K8s.GetKubeArmorNodes()

			for _, nodeIP := range newNodes {
				server := nodeIP + ":" + rs.Port

				if _, ok := ClientList[nodeIP]; ok {
					if !ClientList[nodeIP].Running {
						delete(ClientList, nodeIP)
					}
				} else {
					if client := connectToKubeArmor(server); client != nil {
						ClientList[nodeIP] = client
					}
				}
			}

			time.Sleep(time.Second * 1)
		}
	}
}
