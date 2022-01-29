package core

import (
	"context"
	"math/rand"
	"time"

	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"

	pb "github.com/kubearmor/KubeArmor/protobuf"
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
	alertIn.Filter = "all"

	alertStream, err := lc.client.WatchAlerts(context.Background(), &alertIn)
	if err != nil {
		kg.Warnf("Failed to call WatchAlerts (%s)\n", server)
		return nil
	}
	lc.alertStream = alertStream

	// == //

	logIn := pb.RequestMessage{}
	logIn.Filter = "all"

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
	// generate a nonce
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
	for lc.Running {
		res, err := lc.msgStream.Recv()
		if err != nil {
			if lc.Running {
				kg.Warnf("Failed to receive a message (%s)", lc.server)
				lc.DestroyClient()
				return err
			}
			break
		}

		MsgLock.Lock()
		defer MsgLock.Unlock()

		for uid := range MsgStructs {
			MsgStructs[uid].Queue.Push(*res)
		}
	}

	return nil
}

// WatchAlerts Function
func (lc *LogClient) WatchAlerts() error {
	for lc.Running {
		res, err := lc.alertStream.Recv()
		if err != nil {
			if lc.Running {
				kg.Warnf("Failed to receive an alert (%s)", lc.server)
				lc.DestroyClient()
				return err
			}
			break
		}

		AlertLock.Lock()
		defer AlertLock.Unlock()

		for uid := range AlertStructs {
			AlertStructs[uid].Queue.Push(*res)
		}
	}

	return nil
}

// WatchLogs Function
func (lc *LogClient) WatchLogs() error {
	for lc.Running {
		res, err := lc.logStream.Recv()
		if err != nil {
			if lc.Running {
				kg.Warnf("Failed to receive a log (%s)", lc.server)
				lc.DestroyClient()
				return err
			}
			break
		}

		LogLock.Lock()
		defer LogLock.Unlock()

		for uid := range LogStructs {
			LogStructs[uid].Queue.Push(*res)
		}
	}

	return nil
}

// DestroyClient Function
func (lc *LogClient) DestroyClient() error {
	lc.Running = false

	// wait for a while
	time.Sleep(1 * time.Second)

	if err := lc.conn.Close(); err != nil {
		return err
	}

	return nil
}
