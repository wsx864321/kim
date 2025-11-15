package event

import (
	"context"
	"errors"
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/internal/gateway/conn"
	"github.com/wsx864321/kim/internal/gateway/infra/grpc/session"
	"github.com/wsx864321/kim/internal/gateway/pkg/id"
	"github.com/wsx864321/kim/pkg/log"
	"github.com/wsx864321/kim/pkg/xerr"
	"github.com/wsx864321/kim/pkg/xjson"
	"net"
	"time"
)

// Event 长连接事件
type Event struct {
	sessionCli session.ClientInterface
}

// NewEvent 创建长连接事件处理器
func NewEvent(sessionCli session.ClientInterface) *Event {
	return &Event{
		sessionCli: sessionCli,
	}
}

// OnLogin 处理登录事件
func (e *Event) OnLogin(ctx context.Context, conn net.Conn, payload []byte, gatewayID string) (*sessionpb.Session, error) {
	loginReq := &sessionpb.LoginReq{
		Payload:    payload,
		ConnId:     id.NextID(),
		RemoteAddr: conn.RemoteAddr().String(),
		GatewayId:  gatewayID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loginResp, err := e.sessionCli.Login(ctx, loginReq)
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

func (e *Event) OnConnect(ctx context.Context, conn conn.Connection) error {
	//TODO 需要实现连接建立后的逻辑
	return nil
}

func (e *Event) OnMessage(ctx context.Context, conn conn.Connection, data []byte) error {
	//TODO 需要实现消息转发逻辑
	return nil
}

func (e *Event) OnDisconnect(ctx context.Context, conn conn.Connection, reason string) {
	req := &sessionpb.DelSessionReq{
		UserId:   conn.UserID(),
		DeviceId: []string{conn.DeviceID()},
		Reason:   reason,
	}
	resp, err := e.sessionCli.DelSession(ctx, req)
	if err != nil {
		log.Error(ctx,
			"call session DelSession failed",
			log.String("req", xjson.MarshalString(req)),
			log.String("err", err.Error()))
		return
	}

	if resp.Code != xerr.OK.Code() {
		log.Warn(ctx,
			"session DelSession failed",
			log.String("req", xjson.MarshalString(req)),
			log.Int("code", int(resp.Code)),
			log.String("message", resp.Message))
		return
	}
}

func (e *Event) OnHeartbeat(ctx context.Context, conn conn.Connection) {
	//todo log 暂时不处理心跳事件
}

// OnRefreshSession 刷新会话
func (e *Event) OnRefreshSession(ctx context.Context, conn conn.Connection, lastActiveAt int64) error {
	resp, err := e.sessionCli.RefreshSessionTTL(ctx, &sessionpb.RefreshSessionTTLReq{
		UserId:       conn.UserID(),
		DeviceId:     conn.DeviceID(),
		LastActiveAt: lastActiveAt,
	})
	if err != nil {
		log.Error(ctx, "call session RefreshSessionTTL failed", log.String("err", err.Error()))
		return nil
	}

	if resp.Code != xerr.OK.Code() {
		log.Warn(ctx, "session RefreshSessionTTL failed", log.Int("code", int(resp.Code)), log.String("message", resp.Message))
		return errors.New(resp.Message)
	}

	return nil
}
