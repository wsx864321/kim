package gateway

import (
	"context"
	"errors"
	gatewaypb "github.com/wsx864321/kim/idl/gateway"
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/pkg/log"
	"github.com/wsx864321/kim/pkg/xerr"
	"net"
	"time"
)

type Handler struct {
	sessionClient sessionpb.SessionServiceClient
	transport     Transport

	gatewaypb.UnimplementedGatewayServiceServer
}

// NewHandler 创建 Handler 实例
func NewHandler(sessionClient sessionpb.SessionServiceClient, transport Transport) *Handler {
	return &Handler{
		sessionClient: sessionClient,
		transport:     transport,
	}
}

// OnLogin 鉴权登录
func (h *Handler) OnLogin(ctx context.Context, conn net.Conn, payload []byte, gatewayID string) (*sessionpb.Session, error) {
	// 调用 Session.Login
	loginReq := &sessionpb.LoginReq{
		Payload:    payload,
		ConnId:     connIDGenerator.NextID(),
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

func (h *Handler) OnConnect(ctx context.Context, conn Connection) error {
	//TODO implement me
	panic("implement me")
}

func (h *Handler) OnMessage(ctx context.Context, conn Connection, data []byte) error {
	//TODO implement me
	panic("implement me")
}

func (h *Handler) OnDisconnect(ctx context.Context, conn Connection, reason string) {
	//TODO implement me
	panic("implement me")
}

func (h *Handler) OnHeartbeatTimeout(ctx context.Context, conn Connection) {
	//TODO implement me
	panic("implement me")
}

func (h *Handler) OnHeartbeat(ctx context.Context, conn Connection) {
	//TODO implement me
	panic("implement me")
}

func (h *Handler) OnRefreshSession(ctx context.Context, conn Connection, lastActiveAt int64) error {
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
