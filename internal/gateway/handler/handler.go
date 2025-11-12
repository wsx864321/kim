package handler

import (
	"context"
	"errors"
	"fmt"
	gatewaypb "github.com/wsx864321/kim/idl/gateway"
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/internal/gateway/conn"
	"github.com/wsx864321/kim/internal/gateway/pkg/id"
	"github.com/wsx864321/kim/pkg/log"
	"github.com/wsx864321/kim/pkg/xerr"
	"net"
	"time"
)

type GatewayHandler struct {
	sessionClient sessionpb.SessionServiceClient
	transport     conn.Transport

	gatewaypb.UnimplementedGatewayServiceServer
}

// NewGatewayHandler 创建 Handler 实例
func NewGatewayHandler(sessionClient sessionpb.SessionServiceClient, transport conn.Transport) *GatewayHandler {
	return &GatewayHandler{
		sessionClient: sessionClient,
		transport:     transport,
	}
}

// PushMsg 推送消息到指定连接（gRPC接口）
func (h *GatewayHandler) PushMsg(ctx context.Context, req *gatewaypb.PushReq) (*gatewaypb.PushResp, error) {
	if req.ConnId == "" {
		log.Warn(ctx, "conn_id is required")
		return &gatewaypb.PushResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}

	// 解析连接ID
	connID, err := parseConnID(req.ConnId)
	if err != nil {
		log.Warn(ctx, "invalid conn_id", log.String("conn_id", req.ConnId), log.String("error", err.Error()))
		return &gatewaypb.PushResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "invalid conn_id",
		}, nil
	}

	// 通过transport发送消息
	err = h.transport.Send(ctx, connID, req.Msg)
	if err != nil {
		log.Warn(ctx, "push message failed",
			log.String("conn_id", req.ConnId),
			log.String("error", err.Error()),
		)
		return &gatewaypb.PushResp{
			Code:    xerr.ErrInternalServer.Code(),
			Message: err.Error(),
		}, nil
	}

	return &gatewaypb.PushResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
	}, nil
}

// BatchPushMsg 批量推送消息（gRPC接口）
func (h *GatewayHandler) BatchPushMsg(ctx context.Context, req *gatewaypb.BatchPushReq) (*gatewaypb.BatchPushResp, error) {
	if len(req.Targets) == 0 {
		log.Warn(ctx, "targets is empty")
		return &gatewaypb.BatchPushResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "targets is empty",
		}, nil
	}

	// 解析连接ID列表
	connIDs := make([]int, 0, len(req.Targets))
	for _, target := range req.Targets {
		connID, err := parseConnID(target)
		if err != nil {
			log.Warn(ctx, "invalid target", log.String("target", target), log.String("error", err.Error()))
			continue
		}
		connIDs = append(connIDs, connID)
	}

	if len(connIDs) == 0 {
		return &gatewaypb.BatchPushResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "no valid targets",
		}, nil
	}

	// 批量发送消息
	failConns, err := h.transport.BatchSend(ctx, connIDs, req.Msg)
	if err != nil {
		log.Warn(ctx, "batch push message failed", log.String("error", err.Error()))
		return &gatewaypb.BatchPushResp{
			Code:    xerr.ErrInternalServer.Code(),
			Message: err.Error(),
		}, nil
	}

	// 构建结果列表
	results := make([]*gatewaypb.PushResult, 0, len(req.Targets))
	failMap := make(map[uint64]bool)
	for _, failConn := range failConns {
		failMap[failConn] = true
	}

	for i, target := range req.Targets {
		if i < len(connIDs) {
			connID := uint64(connIDs[i])
			if failMap[connID] {
				results = append(results, &gatewaypb.PushResult{
					Target:  target,
					Code:    xerr.ErrInternalServer.Code(),
					Message: "send failed",
				})
			} else {
				results = append(results, &gatewaypb.PushResult{
					Target:  target,
					Code:    xerr.OK.Code(),
					Message: xerr.OK.Error(),
				})
			}
		}
	}

	return &gatewaypb.BatchPushResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
		Results: results,
	}, nil
}

// CloseConn 关闭指定连接（gRPC接口）
func (h *GatewayHandler) CloseConn(ctx context.Context, req *gatewaypb.CloseConnReq) (*gatewaypb.CloseConnResp, error) {
	if req.ConnId == "" {
		log.Warn(ctx, "conn_id is required")
		return &gatewaypb.CloseConnResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}

	// 解析连接ID
	connID, err := parseConnID(req.ConnId)
	if err != nil {
		log.Warn(ctx, "invalid conn_id", log.String("conn_id", req.ConnId), log.String("error", err.Error()))
		return &gatewaypb.CloseConnResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "invalid conn_id",
		}, nil
	}

	// 关闭连接
	err = h.transport.CloseConn(ctx, connID)
	if err != nil {
		log.Warn(ctx, "close connection failed",
			log.String("conn_id", req.ConnId),
			log.String("error", err.Error()),
		)
		return &gatewaypb.CloseConnResp{
			Code:    xerr.ErrInternalServer.Code(),
			Message: err.Error(),
		}, nil
	}

	return &gatewaypb.CloseConnResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
	}, nil
}

// parseConnID 解析连接ID字符串为整数
func parseConnID(connIDStr string) (int, error) {
	// 这里可以根据实际的连接ID格式进行解析
	// 如果连接ID是数字字符串，可以直接转换
	// 如果包含其他格式，需要相应处理
	var connID uint64
	_, err := fmt.Sscanf(connIDStr, "%d", &connID)
	if err != nil {
		return 0, err
	}
	return int(connID), nil
}

// OnLogin 鉴权登录
func (h *GatewayHandler) OnLogin(ctx context.Context, conn net.Conn, payload []byte, gatewayID string) (*sessionpb.Session, error) {
	// 调用 Session.Login
	loginReq := &sessionpb.LoginReq{
		Payload:    payload,
		ConnId:     id.NextID(),
		RemoteAddr: conn.RemoteAddr().String(),
		GatewayId:  gatewayID,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loginResp, err := h.sessionClient.Login(ctx, loginReq)
	if err != nil {
		log.Warn(ctx, "call session logic failed", log.String("error", err.Error()), log.String("remote", conn.RemoteAddr().String()))
		return nil, err
	}

	// 检查登录结果
	if loginResp.Code != xerr.OK.Code() {
		log.Warn(ctx, "session logic failed", log.Int("code", int(loginResp.Code)), log.String("message", loginResp.Message), log.String("remote", conn.RemoteAddr().String()))
		return nil, err
	}

	// 登录成功，从响应中获取会话信息
	session := loginResp.GetData().GetSession()
	if session == nil {
		log.Error(ctx, "session data is nil in logic response")
		return nil, errors.New("session data is nil in logic response")
	}

	if session.ConnId != loginReq.ConnId {
		log.Error(ctx, "mismatched conn ID in session data", log.Uint64("expected", loginReq.ConnId), log.Uint64("actual", session.ConnId))
		return nil, errors.New("mismatched conn ID in session data")
	}

	return session, nil
}

// OnConnect 连接建立且已鉴权
func (h *GatewayHandler) OnConnect(ctx context.Context, conn conn.Connection) error {
	log.Info(ctx, "connection established",
		log.Uint64("connID", conn.ID()),
		log.String("userID", conn.UserID()),
		log.String("deviceID", conn.DeviceID()),
		log.String("remoteAddr", conn.RemoteAddr().String()),
	)
	// TODO: 可以在这里添加连接建立后的业务逻辑

	return nil
}

// OnMessage 收到业务消息
func (h *GatewayHandler) OnMessage(ctx context.Context, conn conn.Connection, data []byte) error {
	log.Debug(ctx, "received message",
		log.Uint64("connID", conn.ID()),
		log.String("userID", conn.UserID()),
		log.Int("dataLen", len(data)),
	)
	// TODO: 实现业务消息处理逻辑
	// 消息路由、消息转发等
	return nil
}

// OnDisconnect 连接断开
func (h *GatewayHandler) OnDisconnect(ctx context.Context, conn conn.Connection, reason string) {
	log.Info(ctx, "connection disconnected",
		log.Uint64("connID", conn.ID()),
		log.String("userID", conn.UserID()),
		log.String("deviceID", conn.DeviceID()),
		log.String("reason", reason),
	)
	// TODO: 可以在这里添加连接断开后的业务逻辑
	// 通知其他服务、清理资源等
}

// OnHeartbeat 收到心跳消息
func (h *GatewayHandler) OnHeartbeat(ctx context.Context, conn conn.Connection) {
	log.Debug(ctx, "heartbeat received",
		log.Uint64("connID", conn.ID()),
		log.String("userID", conn.UserID()),
	)
	// TODO: 可以在这里添加心跳处理逻辑
	// 更新统计信息等
}

// OnRefreshSession 刷新会话信息
func (h *GatewayHandler) OnRefreshSession(ctx context.Context, conn conn.Connection, lastActiveAt int64) error {
	resp, err := h.sessionClient.RefreshSessionTTL(ctx, &sessionpb.RefreshSessionTTLReq{
		UserId:       conn.UserID(),
		DeviceId:     conn.DeviceID(),
		LastActiveAt: lastActiveAt,
	})
	if err != nil {
		log.Error(ctx, "call session RefreshSessionTTL failed", log.String("err", err.Error()))
		return err
	}

	if resp.Code != xerr.OK.Code() {
		log.Warn(ctx, "session RefreshSessionTTL failed", log.Int("code", int(resp.Code)), log.String("message", resp.Message))
		return errors.New(resp.Message)
	}

	return nil
}
