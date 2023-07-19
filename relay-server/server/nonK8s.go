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
func (ls *PushLogService) addMsgStruct(uid string, conn chan *pb.Message, filter string) {
	MsgLock.Lock()
	defer MsgLock.Unlock()

	msgStruct := MsgStruct{}
	msgStruct.Filter = filter
	msgStruct.Broadcast = conn

	MsgStructs[uid] = msgStruct

	kg.Printf("Added a new client (%s) for PushMessages", uid)
}

// removeMsgStruct Function
func (ls *PushLogService) removeMsgStruct(uid string) {
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
	ls.addMsgStruct(uid, conn, "all")
	defer ls.removeMsgStruct(uid)

	var err error
	for Running {
		var res *pb.Message

		if res, err = svr.Recv(); err != nil {
			kg.Warnf("Failed to recieve an alert")
		}

		msg := pb.Message{}

		if err := kl.Clone(*res, &msg); err != nil {
			kg.Warnf("Failed to clone an alert (%v)", *res)
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
		/*
		select {
		case <-svr.Context().Done():
			return nil
		case resp := <-conn:
			if status, ok := status.FromError(svr.Recv()); ok {
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
		*/
	}

	kg.Print("Stopped receiving pushed messages from")

	return nil
}

// addAlertStruct Function
func (ls *PushLogService) addAlertStruct(uid string, conn chan *pb.Alert, filter string) {
	AlertLock.Lock()
	defer AlertLock.Unlock()

	alertStruct := AlertStruct{}
	alertStruct.Filter = filter
	alertStruct.Broadcast = conn

	AlertStructs[uid] = alertStruct

	kg.Printf("Added a new client (%s, %s) for PushAlerts", uid, filter)
}

// removeAlertStruct Function
func (ls *PushLogService) removeAlertStruct(uid string) {
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
	ls.addAlertStruct(uid, conn, "policy")
	defer ls.removeAlertStruct(uid)

	var err error
	for Running {
		var res *pb.Alert

		if res, err = svr.Recv(); err != nil {
			kg.Warnf("Failed to recieve an alert")
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

		/*
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
		*/
	}

	kg.Print("Stopped receiving pushed alerts from")

	return nil
}

// addLogStruct Function
func (ls *PushLogService) addLogStruct(uid string, conn chan *pb.Log, filter string) {
	LogLock.Lock()
	defer LogLock.Unlock()

	logStruct := LogStruct{}
	logStruct.Filter = filter
	logStruct.Broadcast = conn

	LogStructs[uid] = logStruct

	kg.Printf("Added a new client (%s, %s) for PushLogs", uid, filter)
}

// removeLogStruct Function
func (ls *PushLogService) removeLogStruct(uid string) {
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
	ls.addLogStruct(uid, conn, "system")
	defer ls.removeLogStruct(uid)

	var err error
	for Running {
		var res *pb.Log

		if res, err = svr.Recv(); err != nil {
			kg.Warnf("Failed to recieve a log")
		}

		log := pb.Log{}

		if err := kl.Clone(*res, &log); err != nil {
			kg.Warnf("Failed to clone a message (%v)", *res)
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
		/*
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
		*/
	}

	kg.Print("Stopped receiving pushed logs from")

	return nil
}
