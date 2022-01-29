package core

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	v1 "k8s.io/api/core/v1"

	kl "github.com/kubearmor/kubearmor-relay-server/relay-server/common"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

// ============ //
// == Global == //
// ============ //

// Running flag
var Running bool

func init() {
	Running = true
}

// ========== //
// == gRPC == //
// ========== //

// MsgStruct Structure
type MsgStruct struct {
	Filter string
	Queue  *kl.Queue
}

// MsgStructs Map
var MsgStructs map[string]MsgStruct

// MsgLock Lock
var MsgLock *sync.RWMutex

// AlertStruct Structure
type AlertStruct struct {
	Filter string
	Queue  *kl.Queue
}

// AlertStructs Map
var AlertStructs map[string]AlertStruct

// AlertLock Lock
var AlertLock *sync.RWMutex

// LogStruct Structure
type LogStruct struct {
	Filter string
	Queue  *kl.Queue
}

// LogStructs Map
var LogStructs map[string]LogStruct

// LogLock Lock
var LogLock *sync.RWMutex

// LogService Structure
type LogService struct {
	//
}

// HealthCheck Function
func (ls *LogService) HealthCheck(ctx context.Context, nonce *pb.NonceMessage) (*pb.ReplyMessage, error) {
	replyMessage := pb.ReplyMessage{Retval: nonce.Nonce}
	return &replyMessage, nil
}

// addMsgStruct Function
func (ls *LogService) addMsgStruct(uid string, filter string) {
	MsgLock.Lock()
	defer MsgLock.Unlock()

	msgStruct := MsgStruct{}
	msgStruct.Filter = filter
	msgStruct.Queue = kl.NewQueue()

	MsgStructs[uid] = msgStruct

	kg.Printf("Added a new client (%s) for WatchMessages", uid)
}

// removeMsgStruct Function
func (ls *LogService) removeMsgStruct(uid string) {
	MsgLock.Lock()
	defer MsgLock.Unlock()

	delete(MsgStructs, uid)

	kg.Printf("Deleted the client (%s) for WatchMessages", uid)
}

// WatchMessages Function
func (ls *LogService) WatchMessages(req *pb.RequestMessage, svr pb.LogService_WatchMessagesServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()

	ls.addMsgStruct(uid, req.Filter)
	defer ls.removeMsgStruct(uid)

	for Running {
		select {
		case <-svr.Context().Done():
			return nil
		default:
			if msgInt := MsgStructs[uid].Queue.Pop(); msgInt != nil {
				msg := msgInt.(pb.Message)
				if status, ok := status.FromError(svr.Send(&msg)); ok {
					switch status.Code() {
					case codes.OK:
						// noop
					case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
						kg.Warnf("Failed to send a message=[%+v] err=[%s]", msg, status.Err().Error())
						return status.Err()
					default:
						return nil
					}
				}
			} else {
				time.Sleep(time.Second * 1)
			}
		}
	}

	return nil
}

// addAlertStruct Function
func (ls *LogService) addAlertStruct(uid string, filter string) {
	AlertLock.Lock()
	defer AlertLock.Unlock()

	alertStruct := AlertStruct{}
	alertStruct.Filter = filter
	alertStruct.Queue = kl.NewQueue()

	AlertStructs[uid] = alertStruct

	kg.Printf("Added a new client (%s, %s) for WatchAlerts", uid, filter)
}

// removeAlertStruct Function
func (ls *LogService) removeAlertStruct(uid string) {
	AlertLock.Lock()
	defer AlertLock.Unlock()

	delete(AlertStructs, uid)

	kg.Printf("Deleted the client (%s) for WatchAlerts", uid)
}

// WatchAlerts Function
func (ls *LogService) WatchAlerts(req *pb.RequestMessage, svr pb.LogService_WatchAlertsServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()

	if req.Filter != "all" && req.Filter != "policy" {
		return nil
	}

	ls.addAlertStruct(uid, req.Filter)
	defer ls.removeAlertStruct(uid)

	for Running {
		select {
		case <-svr.Context().Done():
			return nil
		default:
			if alertInt := AlertStructs[uid].Queue.Pop(); alertInt != nil {
				alert := alertInt.(pb.Alert)
				if status, ok := status.FromError(svr.Send(&alert)); ok {
					switch status.Code() {
					case codes.OK:
						// noop
					case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
						kg.Warnf("Failed to send an alert=[%+v] err=[%s]", alert, status.Err().Error())
						return status.Err()
					default:
						return nil
					}
				}
			} else {
				time.Sleep(time.Second * 1)
			}
		}
	}

	return nil
}

// addLogStruct Function
func (ls *LogService) addLogStruct(uid string, filter string) {
	LogLock.Lock()
	defer LogLock.Unlock()

	logStruct := LogStruct{}
	logStruct.Filter = filter
	logStruct.Queue = kl.NewQueue()

	LogStructs[uid] = logStruct

	kg.Printf("Added a new client (%s, %s) for WatchLogs", uid, filter)
}

// removeLogStruct Function
func (ls *LogService) removeLogStruct(uid string) {
	LogLock.Lock()
	defer LogLock.Unlock()

	delete(LogStructs, uid)

	kg.Printf("Deleted the client (%s) for WatchLogs", uid)
}

// WatchLogs Function
func (ls *LogService) WatchLogs(req *pb.RequestMessage, svr pb.LogService_WatchLogsServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()

	if req.Filter != "all" && req.Filter != "system" {
		return nil
	}

	ls.addLogStruct(uid, req.Filter)
	defer ls.removeLogStruct(uid)

	for Running {
		select {
		case <-svr.Context().Done():
			return nil
		default:
			if logInt := LogStructs[uid].Queue.Pop(); logInt != nil {
				log := logInt.(pb.Log)
				if status, ok := status.FromError(svr.Send(&log)); ok {
					switch status.Code() {
					case codes.OK:
						// noop
					case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
						kg.Warnf("Failed to send a log=[%+v] err=[%s]", log, status.Err().Error())
						return status.Err()
					default:
						return nil
					}
				}
			} else {
				time.Sleep(time.Second * 1)
			}
		}
	}

	return nil
}

// ================== //
// == Relay Server == //
// ================== //

// RelayServer Structure
type RelayServer struct {
	// port
	Port string

	// gRPC listener
	Listener net.Listener

	// log server
	LogServer *grpc.Server

	// client list
	ClientList map[string]*LogClient

	// wait group
	WgServer sync.WaitGroup
}

// K8sPodEvent Structure
type K8sPodEvent struct {
	Type   string `json:"type"`
	Object v1.Pod `json:"object"`
}

// NewRelayServer Function
func NewRelayServer(port string) *RelayServer {
	rs := &RelayServer{}

	rs.Port = port

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
		Timeout: 1 * time.Second,
	}

	// create a log server
	rs.LogServer = grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))

	// register a log service
	logService := &LogService{}
	pb.RegisterLogServiceServer(rs.LogServer, logService)

	// initialize msg structs
	MsgStructs = make(map[string]MsgStruct)
	MsgLock = &sync.RWMutex{}

	// initialize alert structs
	AlertStructs = make(map[string]AlertStruct)
	AlertLock = &sync.RWMutex{}

	// initialize log structs
	LogStructs = make(map[string]LogStruct)
	LogLock = &sync.RWMutex{}

	// reset a client list
	rs.ClientList = map[string]*LogClient{}

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
		rs.Listener.Close()
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
	rs.LogServer.Serve(rs.Listener)
}

// =============== //
// == KubeArmor == //
// =============== //

// ConnectToKubeArmor Function
func (rs *RelayServer) ConnectToKubeArmor(server string) *LogClient {
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

				if _, ok := rs.ClientList[nodeIP]; !ok {
					if client := rs.ConnectToKubeArmor(server); client != nil {
						rs.ClientList[nodeIP] = client
					}
				} else if !rs.ClientList[nodeIP].Running {
					delete(rs.ClientList, nodeIP)
				}
			}

			time.Sleep(time.Second * 1)
		}
	}
}
