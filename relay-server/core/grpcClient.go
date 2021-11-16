package core

import (
	"context"
	"math/rand"

	pb "github.com/kubearmor/KubeArmor/protobuf"
	"github.com/kubearmor/kubearmor-relay-server/relay-server/log"
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
		log.Warnf("Failed to connect to KubeArmor's gRPC service (%s)\n", server)
		return nil
	}
	lc.conn = conn

	lc.client = pb.NewLogServiceClient(lc.conn)

	// == //

	msgIn := pb.RequestMessage{}
	msgIn.Filter = ""

	msgStream, err := lc.client.WatchMessages(context.Background(), &msgIn)
	if err != nil {
		log.Warnf("Failed to call WatchMessages (%s)\n", server)
		return nil
	}
	lc.msgStream = msgStream

	// == //

	alertIn := pb.RequestMessage{}
	alertIn.Filter = ""

	alertStream, err := lc.client.WatchAlerts(context.Background(), &alertIn)
	if err != nil {
		log.Warnf("Failed to call WatchAlerts (%s)\n", server)
		return nil
	}
	lc.alertStream = alertStream

	// == //

	logIn := pb.RequestMessage{}
	logIn.Filter = ""

	logStream, err := lc.client.WatchLogs(context.Background(), &logIn)
	if err != nil {
		log.Warnf("Failed to call WatchLogs (%s)\n", server)
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
		log.Warnf("Failed to check the liveness of KubeArmor's gRPC service (%s)", lc.server)
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
				log.Warnf("Failed to receive a message from KubeArmor's gRPC service (%s)", lc.server)
				lc.DestroyClient()
				return err
			}
			break
		}

		MsgQueue <- *res
	}

	return nil
}

// WatchAlerts Function
func (lc *LogClient) WatchAlerts() error {
	for lc.Running {
		res, err := lc.alertStream.Recv()
		if err != nil {
			if lc.Running {
				log.Warnf("Failed to receive a log from KubeArmor's gRPC service (%s)", lc.server)
				lc.DestroyClient()
				return err
			}
			break
		}

		AlertQueue <- *res
	}

	return nil
}

// WatchLogs Function
func (lc *LogClient) WatchLogs() error {
	for lc.Running {
		res, err := lc.logStream.Recv()
		if err != nil {
			if lc.Running {
				log.Warnf("Failed to receive a log from KubeArmor's gRPC service (%s)", lc.server)
				lc.DestroyClient()
				return err
			}
			break
		}

		LogQueue <- *res
	}

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
