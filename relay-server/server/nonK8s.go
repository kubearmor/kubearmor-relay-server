package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	kl "github.com/kubearmor/kubearmor-relay-server/relay-server/common"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

type PushLogService struct {
}

// HealthCheck Function
func (ls *PushLogService) HealthCheck(ctx context.Context, nonce *pb.NonceMessage) (*pb.ReplyMessage, error) {
	replyMessage := pb.ReplyMessage{Retval: nonce.Nonce}
	return &replyMessage, nil
}

// addMsgStruct Function
func addMsgStruct(uid string, conn chan *pb.Message, filter string) {
	MsgLock.Lock()
	defer MsgLock.Unlock()

	msgStruct := MsgStruct{}
	msgStruct.Filter = filter
	msgStruct.Broadcast = conn

	MsgStructs[uid] = msgStruct

	kg.Printf("Added a new client (%s) for PushMessages", uid)
}

// removeMsgStruct Function
func removeMsgStruct(uid string) {
	MsgLock.Lock()
	defer MsgLock.Unlock()

	delete(MsgStructs, uid)

	kg.Printf("Deleted the client (%s) for PushMessages", uid)
}

// PushMessages Function
func (ls *PushLogService) PushMessages(svr pb.PushLogService_PushMessagesServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()
	conn := make(chan *pb.Message, 1)
	defer close(conn)
	addMsgStruct(uid, conn, "all")
	defer removeMsgStruct(uid)

	var err error
	for Running {
		var res *pb.Message

		select {
		case <-svr.Context().Done():
			replyMessage := pb.ReplyMessage{
				Retval: 1,
			}
			svr.Send(&replyMessage)
			kg.Printf("Closing Message stream for (%s)", uid)
			return nil
		default:
			if res, err = svr.Recv(); err != nil {
				kg.Warnf("Failed to recieve a message")
				return err
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
	}

	kg.Printf("Stopped receiving Pushed messages from (%s)", uid)

	return nil
}

// addAlertStruct Function
func addAlertStruct(uid string, conn chan *pb.Alert, filter string) {
	AlertLock.Lock()
	defer AlertLock.Unlock()

	alertStruct := AlertStruct{}
	alertStruct.Filter = filter
	alertStruct.Broadcast = conn

	AlertStructs[uid] = alertStruct

	kg.Printf("Added a new client (%s, %s) for PushAlerts", uid, filter)
}

// removeAlertStruct Function
func removeAlertStruct(uid string) {
	AlertLock.Lock()
	defer AlertLock.Unlock()

	delete(AlertStructs, uid)

	kg.Printf("Deleted the client (%s) for PushAlerts", uid)
}

// PushAlerts Function
func (ls *PushLogService) PushAlerts(svr pb.PushLogService_PushAlertsServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()

	conn := make(chan *pb.Alert, 1)
	defer close(conn)
	addAlertStruct(uid, conn, "policy")
	defer removeAlertStruct(uid)

	var err error
	for Running {
		var res *pb.Alert

		select {
		case <-svr.Context().Done():
			replyMessage := pb.ReplyMessage{
				Retval: 1,
			}
			svr.Send(&replyMessage)
			kg.Printf("Closing Alert stream for (%s)", uid)
			return nil
		default:

			if res, err = svr.Recv(); err != nil {
				kg.Warnf("Failed to recieve an alert")
				return err
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
	}

	kg.Print("Stopped receiving pushed alerts from")

	return nil
}

// addLogStruct Function
func addLogStruct(uid string, conn chan *pb.Log, filter string) {
	LogLock.Lock()
	defer LogLock.Unlock()

	logStruct := LogStruct{}
	logStruct.Filter = filter
	logStruct.Broadcast = conn

	LogStructs[uid] = logStruct

	kg.Printf("Added a new client (%s, %s) for PushLogs", uid, filter)
}

// removeLogStruct Function
func removeLogStruct(uid string) {
	LogLock.Lock()
	defer LogLock.Unlock()

	delete(LogStructs, uid)

	kg.Printf("Deleted the client (%s) for PushLogs", uid)
}

// PushLogs Function
func (ls *PushLogService) PushLogs(svr pb.PushLogService_PushLogsServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()

	conn := make(chan *pb.Log, 1)
	defer close(conn)
	addLogStruct(uid, conn, "system")
	defer removeLogStruct(uid)

	var err error
	for Running {
		var res *pb.Log

		select {
		case <-svr.Context().Done():
			replyMessage := pb.ReplyMessage{
				Retval: 1,
			}
			svr.Send(&replyMessage)
			kg.Printf("Closing Log stream for (%s)", uid)
			return nil
		default:
			if res, err = svr.Recv(); err != nil {
				kg.Warnf("Failed to recieve a log")
				return err
			}

			log := pb.Log{}

			if err := kl.Clone(*res, &log); err != nil {
				kg.Warnf("Failed to clone a log (%v)", *res)
				continue
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

	kg.Print("Stopped receiving pushed logs")

	return nil
}
