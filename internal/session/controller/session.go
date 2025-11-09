package controller

import (
	"context"
	sessionpb "github.com/wsx864321/kim/idl/session"
	logic "github.com/wsx864321/kim/internal/session/logic"
	"github.com/wsx864321/kim/pkg/log"
	"github.com/wsx864321/kim/pkg/xerr"
)

type SessionController struct {
	sessionpb.UnimplementedSessionServiceServer

	service *logic.SessionService
}

// NewSessionController 创建 Session 控制器
func NewSessionController(s *logic.SessionService) *SessionController {
	return &SessionController{
		service: s,
	}
}

// Login 用户登录，创建会话
func (s *SessionController) Login(ctx context.Context, req *sessionpb.LoginReq) (*sessionpb.LoginResp, error) {
	return s.service.Login(ctx, req)
}

// GetSessions 获取用户会话列表
func (s *SessionController) GetSessions(ctx context.Context, req *sessionpb.GetSessionsReq) (*sessionpb.GetSessionsResp, error) {
	if req.UserId == "" {
		log.Warn(ctx, "user_id is required")
		return &sessionpb.GetSessionsResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}

	return s.service.GetSessions(ctx, req)
}

// Kick 踢人
func (s *SessionController) Kick(ctx context.Context, req *sessionpb.KickReq) (*sessionpb.KickResp, error) {
	if req.UserId == "" {
		log.Warn(ctx, "user_id is required")
		return &sessionpb.KickResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}

	return s.service.Kick(ctx, req)
}

// RefreshSessionTTL 刷新会话 TTL
func (s *SessionController) RefreshSessionTTL(ctx context.Context, req *sessionpb.RefreshSessionTTLReq) (*sessionpb.RefreshSessionTTLResp, error) {
	if req.UserId == "" {
		log.Warn(ctx, "user_id is required")
		return &sessionpb.RefreshSessionTTLResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}

	if req.DeviceId == "" {
		log.Warn(ctx, "device_id is required")
		return &sessionpb.RefreshSessionTTLResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}
	return s.service.RefreshSessionTTL(ctx, req)
}
