// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of KubeArmor

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	kl "github.com/kubearmor/kubearmor-relay-server/relay-server/common"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

// ============ //
// == Global == //
// ============ //

// Running flag
var Running bool

// ClientList Map
var ClientList map[string]int

// ClientListLock Lock
var ClientListLock *sync.Mutex

func init() {
	Running = true
	ClientList = map[string]int{}
	ClientListLock = &sync.Mutex{}

}

// ========== //
// == gRPC == //
// ========== //

// MsgStruct Structure
type MsgStruct struct {
	Filter    string
	Broadcast chan *pb.Message
}

// MsgStructs Map
var MsgStructs map[string]MsgStruct

// MsgLock Lock
var MsgLock *sync.RWMutex

// AlertStruct Structure
type AlertStruct struct {
	Filter    string
	Broadcast chan *pb.Alert
}

// AlertStructs Map
var AlertStructs map[string]AlertStruct

// AlertLock Lock
var AlertLock *sync.RWMutex

// LogStruct Structure
type LogStruct struct {
	Filter    string
	Broadcast chan *pb.Log
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
func (ls *LogService) addMsgStruct(uid string, conn chan *pb.Message, filter string) {
	MsgLock.Lock()
	defer MsgLock.Unlock()

	msgStruct := MsgStruct{}
	msgStruct.Filter = filter
	msgStruct.Broadcast = conn

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
	conn := make(chan *pb.Message, 1)
	defer close(conn)
	ls.addMsgStruct(uid, conn, req.Filter)
	defer ls.removeMsgStruct(uid)

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
					kg.Warnf("relay failed to send a message=[%+v] err=[%s]", resp, status.Err().Error())
					return status.Err()
				default:
					return nil
				}
			}
		}
	}

	return nil
}

// addAlertStruct Function
func (ls *LogService) addAlertStruct(uid string, conn chan *pb.Alert, filter string) {
	AlertLock.Lock()
	defer AlertLock.Unlock()

	alertStruct := AlertStruct{}
	alertStruct.Filter = filter
	alertStruct.Broadcast = conn

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
	conn := make(chan *pb.Alert, 1)
	defer close(conn)
	ls.addAlertStruct(uid, conn, req.Filter)
	defer ls.removeAlertStruct(uid)

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
					kg.Warnf("relay failed to send an alert=[%+v] err=[%s]", resp, status.Err().Error())
					return status.Err()
				default:
					return nil
				}
			}
		}
	}

	return nil
}

// addLogStruct Function
func (ls *LogService) addLogStruct(uid string, conn chan *pb.Log, filter string) {
	LogLock.Lock()
	defer LogLock.Unlock()

	logStruct := LogStruct{}
	logStruct.Filter = filter
	logStruct.Broadcast = conn

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
	conn := make(chan *pb.Log, 1)
	defer close(conn)
	ls.addLogStruct(uid, conn, req.Filter)
	defer ls.removeLogStruct(uid)

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
					kg.Warnf("relay failed to send a log=[%+v] err=[%s]", resp, status.Err().Error())
					return status.Err()
				default:
					return nil
				}
			}
		}
	}

	return nil
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

	// wait group
	WgServer sync.WaitGroup
}

// NewClient Function
func NewClient(server string) *LogClient {
	var err error

	lc := &LogClient{}

	lc.Running = true

	// == //

	lc.server = server

	lc.conn, err = grpc.Dial(lc.server, grpc.WithInsecure())
	if err != nil {
		kg.Warnf("Failed to connect to KubeArmor's gRPC service (%s)", server)
		return nil
	}
	defer func() {
		if err != nil {
			err = lc.DestroyClient()
			if err != nil {
				kg.Warnf("DestroyClient() failed err=%s", err.Error())
			}
		}
	}()

	lc.client = pb.NewLogServiceClient(lc.conn)

	// == //

	msgIn := pb.RequestMessage{}
	msgIn.Filter = "all"

	lc.msgStream, err = lc.client.WatchMessages(context.Background(), &msgIn)
	if err != nil {
		kg.Warnf("Failed to call WatchMessages (%s) err=%s\n", server, err.Error())
		return nil
	}

	// == //

	alertIn := pb.RequestMessage{}
	alertIn.Filter = "policy"

	lc.alertStream, err = lc.client.WatchAlerts(context.Background(), &alertIn)
	if err != nil {
		kg.Warnf("Failed to call WatchAlerts (%s) err=%s\n", server, err.Error())
		return nil
	}

	// == //

	logIn := pb.RequestMessage{}
	logIn.Filter = "system"

	lc.logStream, err = lc.client.WatchLogs(context.Background(), &logIn)
	if err != nil {
		kg.Warnf("Failed to call WatchLogs (%s)\n err=%s", server, err.Error())
		return nil
	}

	// == //

	// set wait group
	lc.WgServer = sync.WaitGroup{}

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

	defer lc.WgServer.Done()

	var err error

	for lc.Running {
		var res *pb.Message

		if res, err = lc.msgStream.Recv(); err != nil {
			kg.Warnf("Failed to receive a message (%s)", lc.server)
			break
		}

		msg := pb.Message{}

		if err := kl.Clone(*res, &msg); err != nil {
			kg.Warnf("Failed to clone a message (%v)", *res)
			continue
		}

		tel, _ := json.Marshal(msg)
		fmt.Printf("%s\n", string(tel))

		MsgLock.RLock()
		for uid := range MsgStructs {
			select {
			case MsgStructs[uid].Broadcast <- (&msg):
			default:
			}
		}
		MsgLock.RUnlock()
	}

	kg.Print("Stopped watching messages from " + lc.server)

	return nil
}

// WatchAlerts Function
func (lc *LogClient) WatchAlerts() error {

	defer lc.WgServer.Done()

	var err error

	for lc.Running {
		var res *pb.Alert

		if res, err = lc.alertStream.Recv(); err != nil {
			kg.Warnf("Failed to receive an alert (%s)", lc.server)
			break
		}

		alert := pb.Alert{}

		if err := kl.Clone(*res, &alert); err != nil {
			kg.Warnf("Failed to clone an alert (%v)", *res)
			continue
		}

		tel, _ := json.Marshal(alert)
		fmt.Printf("%s\n", string(tel))

		AlertLock.RLock()
		for uid := range AlertStructs {
			select {
			case AlertStructs[uid].Broadcast <- (&alert):
			default:
			}
		}
		AlertLock.RUnlock()
	}

	kg.Print("Stopped watching alerts from " + lc.server)

	return nil
}

// WatchLogs Function
func (lc *LogClient) WatchLogs() error {

	defer lc.WgServer.Done()

	var err error

	for lc.Running {
		var res *pb.Log

		if res, err = lc.logStream.Recv(); err != nil {
			kg.Warnf("Failed to receive a log (%s)", lc.server)
			break
		}

		log := pb.Log{}

		if err := kl.Clone(*res, &log); err != nil {
			kg.Warnf("Failed to clone a log (%v)", *res)
		}

		tel, _ := json.Marshal(log)
		fmt.Printf("%s\n", string(tel))

		LogLock.RLock()
		for uid := range LogStructs {
			select {
			case LogStructs[uid].Broadcast <- (&log):
			default:
			}
		}
		LogLock.RUnlock()
	}

	kg.Print("Stopped watching logs from " + lc.server)

	return nil
}

// DestroyClient Function
func (lc *LogClient) DestroyClient() error {
	lc.Running = false

	if err := lc.conn.Close(); err != nil {
		return err
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

	// wait group
	WgServer sync.WaitGroup
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
	if err := rs.LogServer.Serve(rs.Listener); err != nil {
		kg.Print("Terminated the gRPC service")
	}
}

func (rs *RelayServer) ListenOnHTTP() {
	http.HandleFunc("/", mainHandler)

	log.Printf("[INFO]  : Falco Sidekick is up and listening on %s:%d", "", "2801")

	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", "", 2801),
		// Timeouts
		ReadTimeout:       60 * time.Second,
		ReadHeaderTimeout: 60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[ERROR] : %v", err.Error())
	}

}

// mainHandler is Falco Sidekick main handler (default).
func mainHandler(w http.ResponseWriter, r *http.Request) {

	if r.Body == nil {
		http.Error(w, "Please send a valid request body", http.StatusBadRequest)

		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Please send with post http method", http.StatusBadRequest)

		return
	}

	d := json.NewDecoder(r.Body)
	d.UseNumber()

	logType := r.Header["Telemetry"][0]

	if logType == "Alert" {
		alert := pb.Alert{}

		if err := d.Decode(&alert); err != nil {
			kg.Warnf("Failed to clone an alert (%v)", err)
			return
		}

		tel, _ := json.Marshal(alert)
		fmt.Printf("%s\n", string(tel))

		AlertLock.RLock()
		for uid := range AlertStructs {
			select {
			case AlertStructs[uid].Broadcast <- (&alert):
			default:
			}
		}
		AlertLock.RUnlock()
	} else if logType == "Log" {
		log := pb.Log{}

		if err := d.Decode(&log); err != nil {
			kg.Warnf("Failed to clone an alert (%v)", err)
			return
		}

		tel, _ := json.Marshal(log)
		fmt.Printf("%s\n", string(tel))

		LogLock.RLock()
		for uid := range LogStructs {
			select {
			case LogStructs[uid].Broadcast <- (&log):
			default:
			}
		}
		LogLock.RUnlock()
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

func connectToKubeArmor(nodeIP, port string) error {

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
func (rs *RelayServer) GetFeedsFromNodes() {

	rs.WgServer.Add(1)
	defer rs.WgServer.Done()

	if K8s.InitK8sClient() {
		kg.Print("Initialized the Kubernetes client")

		for Running {
			newNodes := K8s.GetKubeArmorNodes()

			for _, nodeIP := range newNodes {
				ClientListLock.Lock()
				if _, ok := ClientList[nodeIP]; !ok {
					ClientList[nodeIP] = 1
					go connectToKubeArmor(nodeIP, rs.Port)
				}
				ClientListLock.Unlock()
			}

			time.Sleep(time.Second * 1)
		}

	}
}
