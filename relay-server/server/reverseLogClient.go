// SPDX-License-Identifier: Apache-2.0
// Copyright 2023 Authors of KubeArmor

package server

import (
	"github.com/google/uuid"
	kc "github.com/kubearmor/KubeArmor/KubeArmor/common"
	kf "github.com/kubearmor/KubeArmor/KubeArmor/feeder"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	krc "github.com/kubearmor/kubearmor-relay-server/relay-server/common"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
)

type ReverseLogService struct {
	EventStructs *kf.EventStructs
}

// PushMessages Function
func (ls *ReverseLogService) PushMessages(svr pb.ReverseLogService_PushMessagesServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()

	kg.Printf("Creating PushMessages stream for KubeArmor (%s)", uid)
	defer kg.Printf("Closed PushMessages stream for KubeArmor (%s)", uid)

	var err error
	for Running {
		var res *pb.Message

		select {
		case <-svr.Context().Done():
			replyMessage := pb.ReplyMessage{
				Retval: 1,
			}
			svr.Send(&replyMessage)
			kg.Printf("Closing PushMessages stream for KubeArmor (%s)", uid)
			return nil
		default:
			res, err = svr.Recv()
			if grpcErr := kc.HandleGRPCErrors(err); err != nil {
				kg.Warnf("Failed to recieve a message. Err: %s", grpcErr.Error())
				return err
			}

			msg := pb.Message{}

			if err := krc.Clone(*res, &msg); err != nil {
				kg.Warnf("Failed to clone a message (%v)", *res)
				continue
			}

			ls.EventStructs.MsgLock.RLock()
			for uid := range ls.EventStructs.MsgStructs {
				select {
				case ls.EventStructs.MsgStructs[uid].Broadcast <- (&msg):
				default:
				}
			}
			ls.EventStructs.MsgLock.RUnlock()
		}
	}

	return nil
}

// PushAlerts Function
func (ls *ReverseLogService) PushAlerts(svr pb.ReverseLogService_PushAlertsServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()

	kg.Printf("Creating PushAlerts stream for KubeArmor (%s)", uid)
	defer kg.Printf("Closed PushAlerts stream for KubeArmor (%s)", uid)

	var err error
	for Running {
		var res *pb.Alert

		select {
		case <-svr.Context().Done():
			replyMessage := pb.ReplyMessage{
				Retval: 1,
			}
			svr.Send(&replyMessage)
			kg.Printf("Closing PushAlerts stream for KubeArmor (%s)", uid)
			return nil
		default:
			res, err = svr.Recv()
			if grpcErr := kc.HandleGRPCErrors(err); err != nil {
				kg.Warnf("Failed to recieve an alert. Err: %s", grpcErr.Error())
				return err
			}

			alert := pb.Alert{}

			if err := krc.Clone(*res, &alert); err != nil {
				kg.Warnf("Failed to clone an alert (%v)", *res)
				continue
			}

			ls.EventStructs.AlertLock.RLock()
			for uid := range ls.EventStructs.AlertStructs {
				select {
				case ls.EventStructs.AlertStructs[uid].Broadcast <- (&alert):
				default:
				}
			}
			ls.EventStructs.AlertLock.RUnlock()
		}
	}

	return nil
}

// PushLogs Function
func (ls *ReverseLogService) PushLogs(svr pb.ReverseLogService_PushLogsServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()

	kg.Printf("Creating PushLogs stream for KubeArmor (%s)", uid)
	defer kg.Printf("Closed PushLogs stream for KubeArmor (%s)", uid)

	var err error
	for Running {
		var res *pb.Log

		select {
		case <-svr.Context().Done():
			replyMessage := pb.ReplyMessage{
				Retval: 1,
			}
			svr.Send(&replyMessage)
			kg.Printf("Closing PushLogs stream for KubeArmor (%s)", uid)
			return nil
		default:
			res, err = svr.Recv()
			if grpcErr := kc.HandleGRPCErrors(err); err != nil {
				kg.Warnf("Failed to recieve a log. Err: %s", grpcErr.Error())
				return err
			}

			log := pb.Log{}

			if err := krc.Clone(*res, &log); err != nil {
				kg.Warnf("Failed to clone a log (%v)", *res)
				continue
			}

			ls.EventStructs.LogLock.RLock()
			for uid := range ls.EventStructs.LogStructs {
				select {
				case ls.EventStructs.LogStructs[uid].Broadcast <- (&log):
				default:
				}
			}
			ls.EventStructs.LogLock.RUnlock()
		}
	}

	return nil
}
