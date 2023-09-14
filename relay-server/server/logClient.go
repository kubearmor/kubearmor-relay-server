package server

import (
	"context"
	"math/rand"
	"sync"

	kf "github.com/kubearmor/KubeArmor/KubeArmor/feeder"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	kl "github.com/kubearmor/kubearmor-relay-server/relay-server/common"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
	"google.golang.org/grpc"
)

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

	EventStructs *kf.EventStructs
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

		lc.EventStructs.MsgLock.RLock()
		for uid := range lc.EventStructs.MsgStructs {
			select {
			case lc.EventStructs.MsgStructs[uid].Broadcast <- (&msg):
			default:
			}
		}
		lc.EventStructs.MsgLock.RUnlock()
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

		lc.EventStructs.AlertLock.RLock()
		for uid := range lc.EventStructs.AlertStructs {
			select {
			case lc.EventStructs.AlertStructs[uid].Broadcast <- (&alert):
			default:
			}
		}
		lc.EventStructs.AlertLock.RUnlock()
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

		lc.EventStructs.LogLock.RLock()
		for uid := range lc.EventStructs.LogStructs {
			select {
			case lc.EventStructs.LogStructs[uid].Broadcast <- (&log):
			default:
			}
		}
		lc.EventStructs.LogLock.RUnlock()
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
