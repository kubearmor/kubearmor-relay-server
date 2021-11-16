package core

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	"google.golang.org/grpc"

	v1 "k8s.io/api/core/v1"

	"github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

// ============ //
// == Global == //
// ============ //

// Running flag
var Running bool

// MsgQueue for Messages
var MsgQueue chan pb.Message

// AlertQueue for alerts
var AlertQueue chan pb.Alert

// LogQueue for Logs
var LogQueue chan pb.Log

func init() {
	Running = true

	MsgQueue = make(chan pb.Message, 2048)
	AlertQueue = make(chan pb.Alert, 8192)
	LogQueue = make(chan pb.Log, 65536)
}

// ========== //
// == gRPC == //
// ========== //

// MsgStruct Structure
type MsgStruct struct {
	Client pb.LogService_WatchMessagesServer
	Filter string
}

// AlertStruct Structure
type AlertStruct struct {
	Client pb.LogService_WatchAlertsServer
	Filter string
}

// LogStruct Structure
type LogStruct struct {
	Client pb.LogService_WatchLogsServer
	Filter string
}

// LogService Structure
type LogService struct {
	MsgStructs map[string]MsgStruct
	MsgLock    sync.RWMutex

	AlertStructs map[string]AlertStruct
	AlertLock    sync.RWMutex

	LogStructs map[string]LogStruct
	LogLock    sync.RWMutex
}

// HealthCheck Function
func (ls *LogService) HealthCheck(ctx context.Context, nonce *pb.NonceMessage) (*pb.ReplyMessage, error) {
	replyMessage := pb.ReplyMessage{Retval: nonce.Nonce}
	return &replyMessage, nil
}

// addMsgStruct Function
func (ls *LogService) addMsgStruct(uid string, srv pb.LogService_WatchMessagesServer, filter string) {
	ls.MsgLock.Lock()
	defer ls.MsgLock.Unlock()

	msgStruct := MsgStruct{}
	msgStruct.Client = srv
	msgStruct.Filter = filter

	ls.MsgStructs[uid] = msgStruct
}

// removeMsgStruct Function
func (ls *LogService) removeMsgStruct(uid string) {
	ls.MsgLock.Lock()
	defer ls.MsgLock.Unlock()

	delete(ls.MsgStructs, uid)
}

// getMsgStructs Function
func (ls *LogService) getMsgStructs() []MsgStruct {
	msgStructs := []MsgStruct{}

	ls.MsgLock.RLock()
	defer ls.MsgLock.RUnlock()

	for _, mgs := range ls.MsgStructs {
		msgStructs = append(msgStructs, mgs)
	}

	return msgStructs
}

// WatchMessages Function
func (ls *LogService) WatchMessages(req *pb.RequestMessage, svr pb.LogService_WatchMessagesServer) error {
	var err error
	uid := uuid.Must(uuid.NewRandom()).String()

	ls.addMsgStruct(uid, svr, req.Filter)
	defer ls.removeMsgStruct(uid)

	for Running {
		msg := <-MsgQueue

		msgStructs := ls.getMsgStructs()
		for _, mgs := range msgStructs {
			if err = mgs.Client.Send(&msg); err != nil {
				log.Warn("Failed to send a message")
				break
			}
		}

		if err != nil {
			break
		}
	}

	return nil
}

// addAlertStruct Function
func (ls *LogService) addAlertStruct(uid string, srv pb.LogService_WatchAlertsServer, filter string) {
	ls.AlertLock.Lock()
	defer ls.AlertLock.Unlock()

	alertStruct := AlertStruct{}
	alertStruct.Client = srv
	alertStruct.Filter = filter

	ls.AlertStructs[uid] = alertStruct
}

// removeAlertStruct Function
func (ls *LogService) removeAlertStruct(uid string) {
	ls.AlertLock.Lock()
	defer ls.AlertLock.Unlock()

	delete(ls.AlertStructs, uid)
}

// getAlertStructs Function
func (ls *LogService) getAlertStructs() []AlertStruct {
	alertStructs := []AlertStruct{}

	ls.AlertLock.RLock()
	defer ls.AlertLock.RUnlock()

	for _, als := range ls.AlertStructs {
		alertStructs = append(alertStructs, als)
	}

	return alertStructs
}

// WatchAlerts Function
func (ls *LogService) WatchAlerts(req *pb.RequestMessage, svr pb.LogService_WatchAlertsServer) error {
	var err error
	uid := uuid.Must(uuid.NewRandom()).String()

	ls.addAlertStruct(uid, svr, req.Filter)
	defer ls.removeAlertStruct(uid)

	for Running {
		alert := <-AlertQueue

		alertStructs := ls.getAlertStructs()
		for _, als := range alertStructs {
			if err = als.Client.Send(&alert); err != nil {
				log.Warn("Failed to send an alert")
				break
			}
		}

		if err != nil {
			break
		}
	}

	return nil
}

// addLogStruct Function
func (ls *LogService) addLogStruct(uid string, srv pb.LogService_WatchLogsServer, filter string) {
	ls.LogLock.Lock()
	defer ls.LogLock.Unlock()

	logStruct := LogStruct{}
	logStruct.Client = srv
	logStruct.Filter = filter

	ls.LogStructs[uid] = logStruct
}

// removeLogStruct Function
func (ls *LogService) removeLogStruct(uid string) {
	ls.LogLock.Lock()
	defer ls.LogLock.Unlock()

	delete(ls.LogStructs, uid)
}

// getLogStructs Function
func (ls *LogService) getLogStructs() []LogStruct {
	logStructs := []LogStruct{}

	ls.LogLock.RLock()
	defer ls.LogLock.RUnlock()

	for _, lgs := range ls.LogStructs {
		logStructs = append(logStructs, lgs)
	}

	return logStructs
}

// WatchLogs Function
func (ls *LogService) WatchLogs(req *pb.RequestMessage, svr pb.LogService_WatchLogsServer) error {
	var err error
	uid := uuid.Must(uuid.NewRandom()).String()

	ls.addLogStruct(uid, svr, req.Filter)
	defer ls.removeLogStruct(uid)

	for Running {
		lg := <-LogQueue

		logStructs := ls.getLogStructs()
		for _, lgs := range logStructs {
			if err = lgs.Client.Send(&lg); err != nil {
				log.Warn("Failed to send a log")
				break
			}
		}

		if err != nil {
			break
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
		log.Errf("Failed to listen a port (%s)\n", rs.Port)
		// log.Err(err.Error())
		return nil
	}
	rs.Listener = listener

	// create a log server
	rs.LogServer = grpc.NewServer()

	// register a log service
	logService := &LogService{
		MsgStructs:   make(map[string]MsgStruct),
		MsgLock:      sync.RWMutex{},
		AlertStructs: make(map[string]AlertStruct),
		AlertLock:    sync.RWMutex{},
		LogStructs:   make(map[string]LogStruct),
		LogLock:      sync.RWMutex{},
	}
	pb.RegisterLogServiceServer(rs.LogServer, logService)

	// reset a client list
	rs.ClientList = map[string]*LogClient{}

	// set wait group
	rs.WgServer = sync.WaitGroup{}

	return rs
}

// ServeLogFeeds Function
func (rs *RelayServer) ServeLogFeeds() {
	rs.WgServer.Add(1)
	defer rs.WgServer.Done()

	// feed logs
	rs.LogServer.Serve(rs.Listener)
}

func createClient(server string) *LogClient {
	// create a client
	client := NewClient(server)
	if client == nil {
		log.Errf("Failed to connect to KubeArmor's gRPC service (%s)\n", server)
		return nil
	}

	// do healthcheck
	if ok := client.DoHealthCheck(); !ok {
		log.Warnf("Failed to check the liveness of KubeArmor's gRPC service (%s)", server)
		return nil
	}
	log.Printf("Checked the liveness of KubeArmor's gRPC service (%s)", server)

	// watch messages
	go client.WatchMessages()
	log.Print("Started to watch messages from " + server)

	// watch alerts
	go client.WatchAlerts()
	log.Print("Started to watch alerts from " + server)

	// watch logs
	go client.WatchLogs()
	log.Print("Started to watch logs from " + server)

	return client
}

// GetFeedsFromNodes Function
func (rs *RelayServer) GetFeedsFromNodes() {
	rs.WgServer.Add(1)
	defer rs.WgServer.Done()

	if K8s.InitK8sClient() {
		log.Print("Initialized the Kubernetes client")

		for Running {
			newNodes := K8s.GetKubeArmorNodes()

			for _, nodeIP := range newNodes {
				server := nodeIP + ":" + rs.Port

				if _, ok := rs.ClientList[nodeIP]; !ok {
					client := createClient(server)
					if client != nil {
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
