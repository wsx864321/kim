package handler

import (
	"context"
	"github.com/wsx864321/kim/pkg/xerr"

	pushpb "github.com/wsx864321/kim/idl/push"
	"github.com/wsx864321/kim/internal/push/logic"
)

// PushHandler Push 服务处理器
type PushHandler struct {
	service *logic.PushService

	pushpb.UnimplementedPushServiceServer
}

// NewPushHandler 创建 PushHandler 实例
func NewPushHandler(service *logic.PushService) *PushHandler {
	return &PushHandler{
		service: service,
	}
}

// PushMsg 推送消息到指定用户
func (h *PushHandler) PushMsg(ctx context.Context, req *pushpb.PushReq) (*pushpb.PushResp, error) {
	if req.UserId == "" {
		return &pushpb.PushResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "user_id is empty",
		}, nil
	}

	resp, err := h.service.PushMsg(ctx, req)
	if err != nil {
		return &pushpb.PushResp{
			Code:    err.Code(),
			Message: err.Error(),
		}, nil
	}
	return resp, nil
}

// BatchPushMsg 批量推送消息
func (h *PushHandler) BatchPushMsg(ctx context.Context, req *pushpb.BatchPushReq) (*pushpb.BatchPushResp, error) {
	if len(req.Targets) == 0 {
		return &pushpb.BatchPushResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "targets is empty",
		}, nil
	}

	if req.Msg == "" {
		return &pushpb.BatchPushResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "msg is required",
		}, nil
	}

	resp, err := h.service.BatchPushMsg(ctx, req)
	if err != nil {
		return &pushpb.BatchPushResp{
			Code:    err.Code(),
			Message: err.Error(),
		}, nil
	}
	return resp, nil
}

// CloseConn 关闭指定连接
func (h *PushHandler) CloseConn(ctx context.Context, req *pushpb.CloseConnReq) (*pushpb.CloseConnResp, error) {
	if req.UserId == "" {
		return &pushpb.CloseConnResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "user_id is empty",
		}, nil
	}

	resp, err := h.service.CloseConn(ctx, req)
	if err != nil {
		return &pushpb.CloseConnResp{
			Code:    err.Code(),
			Message: err.Error(),
		}, nil
	}
	return resp, nil
}
