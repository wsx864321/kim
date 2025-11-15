package logic

import (
	"context"
	gatewaypb "github.com/wsx864321/kim/idl/gateway"
	pushpb "github.com/wsx864321/kim/idl/push"
	sessionpb "github.com/wsx864321/kim/idl/session"
	sessiongrpc "github.com/wsx864321/kim/internal/gateway/infra/grpc/session"
	"github.com/wsx864321/kim/internal/push/infra/grpc/gateway"
	"github.com/wsx864321/kim/pkg/log"
	"github.com/wsx864321/kim/pkg/xerr"
)

// PushService Push 业务逻辑服务
type PushService struct {
	sessionClient sessiongrpc.ClientInterface
	gatewayMgr    *gateway.ClientManager
}

// NewPushService 创建 PushService 实例
func NewPushService(sessionClient sessiongrpc.ClientInterface, gatewayMgr *gateway.ClientManager) *PushService {
	return &PushService{
		sessionClient: sessionClient,
		gatewayMgr:    gatewayMgr,
	}
}

// getSessions 获取用户会话（辅助方法）
func (s *PushService) getSessions(ctx context.Context, userID string, deviceID string) (*sessionpb.GetSessionsResp, error) {
	var deviceIDs []string
	if deviceID != "" {
		deviceIDs = []string{deviceID}
	}
	return s.sessionClient.GetSessions(ctx, &sessionpb.GetSessionsReq{
		UserId:   userID,
		DeviceId: deviceIDs,
	})
}

// PushMsg 推送消息到指定用户
func (s *PushService) PushMsg(ctx context.Context, req *pushpb.PushReq) (*pushpb.PushResp, *xerr.Error) {

	// 获取用户会话
	sessionsResp, err := s.getSessions(ctx, req.UserId, req.DeviceId)
	if err != nil {
		log.Error(ctx, "get sessions failed",
			log.String("user_id", req.UserId),
			log.String("device_id", req.DeviceId),
			log.String("error", err.Error()),
		)
		return nil, xerr.ErrInternalServer.WithMessage(err.Error())
	}

	if sessionsResp.Code != xerr.OK.Code() {
		return &pushpb.PushResp{
			Code:    sessionsResp.Code,
			Message: sessionsResp.Message,
		}, nil
	}

	if len(sessionsResp.Data.Sessions) == 0 {
		log.Warn(ctx, "no sessions found",
			log.String("user_id", req.UserId),
			log.String("device_id", req.DeviceId),
		)
		return &pushpb.PushResp{
			Code:    xerr.ErrSessionNotFound.Code(),
			Message: "no sessions found",
		}, nil
	}

	// 推送消息到所有会话
	var lastErr error
	successCount := 0
	for _, session := range sessionsResp.Data.Sessions {
		// 只推送在线状态的会话
		if session.Status != sessionpb.SessionStatus_SESSION_STATUS_ONLINE {
			continue
		}

		// 获取对应的 Gateway 客户端
		gatewayClient, err := s.gatewayMgr.GetClient(session.GatewayId)
		if err != nil {
			log.Error(ctx, "get gateway client failed",
				log.String("gateway_id", session.GatewayId),
				log.String("error", err.Error()),
			)
			lastErr = err
			continue
		}

		// 调用 Gateway 服务推送消息
		_, err = gatewayClient.PushMsg(ctx, &gatewaypb.PushReq{
			ConnId: session.ConnId,
			Msg:    req.Msg,
		})
		if err != nil {
			log.Warn(ctx, "push message to gateway failed",
				log.String("gateway_id", session.GatewayId),
				log.Uint64("conn_id", session.ConnId),
				log.String("error", err.Error()),
			)
			lastErr = err
			continue
		}

		successCount++
	}

	if successCount == 0 {
		if lastErr != nil {
			return &pushpb.PushResp{
				Code:    xerr.ErrInternalServer.Code(),
				Message: lastErr.Error(),
			}, nil
		}
		return &pushpb.PushResp{
			Code:    xerr.ErrSessionNotFound.Code(),
			Message: "no online sessions found",
		}, nil
	}

	return &pushpb.PushResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
	}, nil
}

// BatchPushMsg 批量推送消息
func (s *PushService) BatchPushMsg(ctx context.Context, req *pushpb.BatchPushReq) (*pushpb.BatchPushResp, *xerr.Error) {
	// 批量获取会话并推送
	results := make([]*pushpb.PushResult, 0, len(req.Targets))
	msgBytes := []byte(req.Msg)

	for _, target := range req.Targets {
		if target.UserId == "" {
			results = append(results, &pushpb.PushResult{
				UserId:   target.UserId,
				DeviceId: target.DeviceId,
				Code:     xerr.ErrInvalidParams.Code(),
				Message:  "user_id is required",
			})
			continue
		}

		// 获取用户会话
		sessionsResp, err := s.getSessions(ctx, target.UserId, target.DeviceId)
		if err != nil {
			log.Error(ctx, "get sessions failed",
				log.String("user_id", target.UserId),
				log.String("device_id", target.DeviceId),
				log.String("error", err.Error()),
			)
			results = append(results, &pushpb.PushResult{
				UserId:   target.UserId,
				DeviceId: target.DeviceId,
				Code:     xerr.ErrInternalServer.Code(),
				Message:  err.Error(),
			})
			continue
		}

		if sessionsResp.Code != xerr.OK.Code() {
			results = append(results, &pushpb.PushResult{
				UserId:   target.UserId,
				DeviceId: target.DeviceId,
				Code:     sessionsResp.Code,
				Message:  sessionsResp.Message,
			})
			continue
		}

		if len(sessionsResp.Data.Sessions) == 0 {
			results = append(results, &pushpb.PushResult{
				UserId:   target.UserId,
				DeviceId: target.DeviceId,
				Code:     xerr.ErrSessionNotFound.Code(),
				Message:  "no sessions found",
			})
			continue
		}

		// 推送消息到所有会话
		var lastErr error
		successCount := 0
		for _, session := range sessionsResp.Data.Sessions {
			// 只推送在线状态的会话
			if session.Status != sessionpb.SessionStatus_SESSION_STATUS_ONLINE {
				continue
			}

			// 获取对应的 Gateway 客户端
			gatewayClient, err := s.gatewayMgr.GetClient(session.GatewayId)
			if err != nil {
				log.Error(ctx, "get gateway client failed",
					log.String("gateway_id", session.GatewayId),
					log.String("error", err.Error()),
				)
				lastErr = err
				continue
			}

			// 调用 Gateway 服务推送消息
			_, err = gatewayClient.PushMsg(ctx, &gatewaypb.PushReq{
				ConnId: session.ConnId,
				Msg:    msgBytes,
			})
			if err != nil {
				log.Warn(ctx, "push message to gateway failed",
					log.String("gateway_id", session.GatewayId),
					log.Uint64("conn_id", session.ConnId),
					log.String("error", err.Error()),
				)
				lastErr = err
				continue
			}

			successCount++
		}

		if successCount == 0 {
			if lastErr != nil {
				results = append(results, &pushpb.PushResult{
					UserId:   target.UserId,
					DeviceId: target.DeviceId,
					Code:     xerr.ErrInternalServer.Code(),
					Message:  lastErr.Error(),
				})
			} else {
				results = append(results, &pushpb.PushResult{
					UserId:   target.UserId,
					DeviceId: target.DeviceId,
					Code:     xerr.ErrSessionNotFound.Code(),
					Message:  "no online sessions found",
				})
			}
		} else {
			results = append(results, &pushpb.PushResult{
				UserId:   target.UserId,
				DeviceId: target.DeviceId,
				Code:     xerr.OK.Code(),
				Message:  xerr.OK.Error(),
			})
		}
	}

	return &pushpb.BatchPushResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
		Results: results,
	}, nil
}

// CloseConn 关闭指定连接
func (s *PushService) CloseConn(ctx context.Context, req *pushpb.CloseConnReq) (*pushpb.CloseConnResp, *xerr.Error) {
	sessionsResp, err := s.sessionClient.GetSessions(ctx, &sessionpb.GetSessionsReq{
		UserId:   req.UserId,
		DeviceId: req.GetDeviceId(),
	})
	if err != nil {
		log.Error(ctx, "get sessions failed",
			log.String("user_id", req.UserId),
			log.String("error", err.Error()),
		)
		return nil, xerr.ErrInternalServer.WithMessage(err.Error())
	}

	if sessionsResp.Code != xerr.OK.Code() {
		return &pushpb.CloseConnResp{
			Code:    sessionsResp.Code,
			Message: sessionsResp.Message,
		}, nil
	}

	if len(sessionsResp.Data.Sessions) == 0 {
		log.Warn(ctx, "no sessions found",
			log.String("user_id", req.UserId),
		)
		return &pushpb.CloseConnResp{
			Code:    xerr.ErrSessionNotFound.Code(),
			Message: "no sessions found",
		}, nil
	}

	// 如果指定了 device_id 列表，只关闭指定设备的连接
	// 否则关闭该用户的所有连接
	deviceIDSet := make(map[string]bool)
	for _, deviceID := range req.DeviceId {
		deviceIDSet[deviceID] = true
	}

	// 按 gateway_id 分组，以便批量关闭
	gatewaySessions := make(map[string][]*sessionpb.Session)
	for _, session := range sessionsResp.Data.Sessions {
		// 如果指定了 device_id，只处理匹配的设备
		if len(deviceIDSet) > 0 && !deviceIDSet[session.DeviceId] {
			continue
		}
		// 只关闭在线状态的会话
		if session.Status != sessionpb.SessionStatus_SESSION_STATUS_ONLINE {
			continue
		}
		gatewaySessions[session.GatewayId] = append(gatewaySessions[session.GatewayId], session)
	}

	if len(gatewaySessions) == 0 {
		return &pushpb.CloseConnResp{
			Code:    xerr.ErrSessionNotFound.Code(),
			Message: "no online sessions found to close",
		}, nil
	}

	// 遍历每个 gateway，关闭对应的连接并删除 session
	var lastErr error
	successCount := 0
	closedSessions := make([]*sessionpb.Session, 0) // 记录成功关闭连接的 session

	for gatewayID, sessions := range gatewaySessions {
		// 获取对应的 Gateway 客户端
		gatewayClient, err := s.gatewayMgr.GetClient(gatewayID)
		if err != nil {
			log.Error(ctx, "get gateway client failed",
				log.String("gateway_id", gatewayID),
				log.String("error", err.Error()),
			)
			lastErr = err
			continue
		}

		// 关闭该 gateway 下的所有连接
		for _, session := range sessions {
			_, err = gatewayClient.CloseConn(ctx, &gatewaypb.CloseConnReq{
				ConnId: session.ConnId,
			})
			if err != nil {
				log.Warn(ctx, "close connection failed",
					log.String("gateway_id", gatewayID),
					log.Uint64("conn_id", session.ConnId),
					log.String("user_id", session.UserId),
					log.String("device_id", session.DeviceId),
					log.String("error", err.Error()),
				)
				lastErr = err
				continue
			}
			closedSessions = append(closedSessions, session)
			successCount++
		}
	}

	if successCount == 0 {
		if lastErr != nil {
			return &pushpb.CloseConnResp{
				Code:    xerr.ErrInternalServer.Code(),
				Message: lastErr.Error(),
			}, nil
		}
		return &pushpb.CloseConnResp{
			Code:    xerr.ErrSessionNotFound.Code(),
			Message: "no connections closed",
		}, nil
	}

	// 删除已关闭连接的 session
	// 收集需要删除的 device_id（所有 session 都是同一个 user_id）
	deviceIDs := make([]string, 0, len(closedSessions))
	for _, session := range closedSessions {
		deviceIDs = append(deviceIDs, session.DeviceId)
	}

	// 批量删除该用户的所有相关 session
	if len(deviceIDs) > 0 {
		delResp, err := s.sessionClient.DelSession(ctx, &sessionpb.DelSessionReq{
			UserId:   req.UserId,
			DeviceId: deviceIDs,
			Reason:   "closed by push service",
		})
		if err != nil {
			log.Warn(ctx, "delete session failed",
				log.String("user_id", req.UserId),
				log.Strings("device_ids", deviceIDs),
				log.String("error", err.Error()),
			)
			// 即使删除 session 失败，连接已经关闭，仍然返回成功
		} else if delResp.Code != xerr.OK.Code() {
			log.Warn(ctx, "delete session failed",
				log.String("user_id", req.UserId),
				log.Strings("device_ids", deviceIDs),
				log.Int("code", int(delResp.Code)),
				log.String("message", delResp.Message),
			)
			// 即使删除 session 失败，连接已经关闭，仍然返回成功
		} else {
			log.Info(ctx, "sessions deleted",
				log.String("user_id", req.UserId),
				log.Strings("device_ids", deviceIDs),
			)
		}
	}

	return &pushpb.CloseConnResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
	}, nil
}
