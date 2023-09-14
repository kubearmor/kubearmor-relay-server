package server

import (
	"context"

	kf "github.com/kubearmor/KubeArmor/KubeArmor/feeder"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LogService Structure
type LogService struct {
	EventStructs *kf.EventStructs
}

// Legacy HealthCheck Function
func (ls *LogService) HealthCheck(ctx context.Context, nonce *pb.NonceMessage) (*pb.ReplyMessage, error) {
	replyMessage := pb.ReplyMessage{Retval: nonce.Nonce}
	return &replyMessage, nil
}

// WatchMessages Function
func (ls *LogService) WatchMessages(req *pb.RequestMessage, svr pb.LogService_WatchMessagesServer) error {
	uid, conn := ls.EventStructs.AddMsgStruct(req.Filter, 1)
	kg.Printf("Added a new client (%s) for WatchMessages", uid)

	defer func() {
		close(conn)
		ls.EventStructs.RemoveMsgStruct(uid)
		kg.Printf("Deleted the client (%s) for WatchMessages", uid)
	}()

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

// WatchAlerts Function
func (ls *LogService) WatchAlerts(req *pb.RequestMessage, svr pb.LogService_WatchAlertsServer) error {
	if req.Filter != "all" && req.Filter != "policy" {
		return nil
	}

	uid, conn := ls.EventStructs.AddAlertStruct(req.Filter, 1)
	kg.Printf("Added a new client (%s, %s) for WatchAlerts", uid, req.Filter)

	defer func() {
		close(conn)
		ls.EventStructs.RemoveAlertStruct(uid)
		kg.Printf("Deleted the client (%s) for WatchAlerts", uid)
	}()

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

// WatchLogs Function
func (ls *LogService) WatchLogs(req *pb.RequestMessage, svr pb.LogService_WatchLogsServer) error {
	if req.Filter != "all" && req.Filter != "system" {
		return nil
	}

	uid, conn := ls.EventStructs.AddLogStruct(req.Filter, 1)
	kg.Printf("Added a new client (%s, %s) for WatchLogs", uid, req.Filter)

	defer func() {
		close(conn)
		ls.EventStructs.RemoveLogStruct(uid)
		kg.Printf("Deleted the client (%s) for WatchLogs", uid)
	}()

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
