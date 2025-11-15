package handler

import (
	"context"
	gatewaypb "github.com/wsx864321/kim/idl/gateway"
	"github.com/wsx864321/kim/internal/gateway/conn"
	"github.com/wsx864321/kim/internal/gateway/infra/grpc/session"
	"github.com/wsx864321/kim/pkg/log"
	"github.com/wsx864321/kim/pkg/xerr"
)

type GatewayHandler struct {
	sessionCli session.ClientInterface
	transport  conn.Transport

	gatewaypb.UnimplementedGatewayServiceServer
}

// NewGatewayHandler 创建 Handler 实例
func NewGatewayHandler(sessionCli session.ClientInterface, transport conn.Transport) *GatewayHandler {
	return &GatewayHandler{
		sessionCli: sessionCli,
		transport:  transport,
	}
}

// PushMsg 推送消息到指定连接（gRPC接口）
func (h *GatewayHandler) PushMsg(ctx context.Context, req *gatewaypb.PushReq) (*gatewaypb.PushResp, error) {
	// 通过transport发送消息
	err := h.transport.Send(ctx, req.GetConnId(), req.Msg)
	if err != nil {
		log.Warn(ctx, "push message failed",
			log.Uint64("conn_id", req.GetConnId()),
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
	if len(req.GetConnIds()) == 0 {
		log.Warn(ctx, "targets is empty")
		return &gatewaypb.BatchPushResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: "targets is empty",
		}, nil
	}

	// 批量发送消息
	failConns, err := h.transport.BatchSend(ctx, req.GetConnIds(), req.Msg)
	if err != nil {
		log.Warn(ctx, "batch push message failed", log.String("error", err.Error()))
		return &gatewaypb.BatchPushResp{
			Code:    xerr.ErrInternalServer.Code(),
			Message: err.Error(),
		}, nil
	}

	// 构建结果列表
	results := make([]*gatewaypb.PushResult, 0, len(req.GetConnIds()))
	failMap := make(map[uint64]bool)
	for _, failConn := range failConns {
		failMap[failConn] = true
	}

	for _, connID := range req.GetConnIds() {
		if failMap[connID] {
			results = append(results, &gatewaypb.PushResult{
				ConnId:  connID,
				Code:    xerr.ErrInternalServer.Code(),
				Message: "send failed",
			})
		} else {
			results = append(results, &gatewaypb.PushResult{
				ConnId:  connID,
				Code:    xerr.OK.Code(),
				Message: xerr.OK.Error(),
			})
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
	// 关闭连接
	err := h.transport.CloseConn(ctx, req.GetConnId())
	if err != nil {
		log.Warn(ctx, "close connection failed",
			log.Uint64("conn_id", req.GetConnId()),
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
