package handler

import (
	"context"
	sessionpb "github.com/wsx864321/kim/idl/session"
	logic "github.com/wsx864321/kim/internal/session/logic"
	"github.com/wsx864321/kim/pkg/log"
	"github.com/wsx864321/kim/pkg/xerr"
	"google.golang.org/protobuf/proto"
)

type SessionHandler struct {
	sessionpb.UnimplementedSessionServiceServer

	service *logic.SessionService
}

// NewSessionHandler 创建 Session 控制器
func NewSessionHandler(s *logic.SessionService) *SessionHandler {
	return &SessionHandler{
		service: s,
	}
}

// Login 用户登录，创建会话
func (s *SessionHandler) Login(ctx context.Context, req *sessionpb.LoginReq) (*sessionpb.LoginResp, error) {
	var auth sessionpb.AuthInfo
	if err := proto.Unmarshal(req.Payload, &auth); err != nil {
		log.Warn(ctx, "unmarshal auth info failed", log.String("err", err.Error()))
		return &sessionpb.LoginResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}

	resp, err := s.service.Login(ctx, &auth, req)
	if err != nil {
		return &sessionpb.LoginResp{
			Code:    xerr.Convert(err).Code(),
			Message: xerr.Convert(err).Error(),
		}, nil
	}

	return &sessionpb.LoginResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
		Data:    resp,
	}, nil
}

// DelSession 删除用户会话
func (s *SessionHandler) DelSession(ctx context.Context, req *sessionpb.DelSessionReq) (*sessionpb.DelSessionResp, error) {
	if req.UserId == "" {
		log.Warn(ctx, "user_id is required")
		return &sessionpb.DelSessionResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "user_id is required",
		}, nil
	}

	err := s.service.DelSession(ctx, req)
	if err != nil {
		return &sessionpb.DelSessionResp{
			Code:    err.Code(),
			Message: err.Error(),
		}, nil
	}

	return &sessionpb.DelSessionResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
	}, nil
}

// GetSessions 获取用户会话列表
func (s *SessionHandler) GetSessions(ctx context.Context, req *sessionpb.GetSessionsReq) (*sessionpb.GetSessionsResp, error) {
	if req.UserId == "" {
		log.Warn(ctx, "user_id is required")
		return &sessionpb.GetSessionsResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}

	resp, err := s.service.GetSessions(ctx, req)
	if err != nil {
		return &sessionpb.GetSessionsResp{
			Code:    err.Code(),
			Message: err.Error(),
		}, nil
	}

	return &sessionpb.GetSessionsResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
		Data:    resp,
	}, nil
}

// Kick 踢人
func (s *SessionHandler) Kick(ctx context.Context, req *sessionpb.KickReq) (*sessionpb.KickResp, error) {
	if req.UserId == "" {
		log.Warn(ctx, "user_id is required")
		return &sessionpb.KickResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}

	err := s.service.Kick(ctx, req)
	if err != nil {
		return &sessionpb.KickResp{
			Code:    err.Code(),
			Message: err.Error(),
		}, nil
	}

	return &sessionpb.KickResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
	}, nil
}

// RefreshSessionTTL 刷新会话 TTL
func (s *SessionHandler) RefreshSessionTTL(ctx context.Context, req *sessionpb.RefreshSessionTTLReq) (*sessionpb.RefreshSessionTTLResp, error) {
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

	err := s.service.RefreshSessionTTL(ctx, req)
	if err != nil {
		return &sessionpb.RefreshSessionTTLResp{
			Code:    err.Code(),
			Message: err.Error(),
		}, nil
	}

	return &sessionpb.RefreshSessionTTLResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
	}, nil
}
