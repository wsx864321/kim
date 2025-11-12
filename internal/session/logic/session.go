package logic

import (
	"context"
	"errors"
	"github.com/golang/protobuf/proto"
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/internal/session/infra/redis"
	"github.com/wsx864321/kim/pkg/log"
	"github.com/wsx864321/kim/pkg/xerr"
	"github.com/wsx864321/kim/pkg/xjson"
	"time"
)

type SessionService struct {
	redis redis.InstanceInterface
}

// NewSessionService 创建 SessionService 实例
func NewSessionService(r redis.InstanceInterface) *SessionService {
	return &SessionService{
		redis: r,
	}
}

// Login 用户登录，创建会话
func (s *SessionService) Login(ctx context.Context, req *sessionpb.LoginReq) (*sessionpb.LoginResp, error) {
	var auth sessionpb.AuthInfo
	if err := proto.Unmarshal(req.Payload, &auth); err != nil {
		log.Warn(ctx, "unmarshal auth info failed", log.String("err", err.Error()))
		return &sessionpb.LoginResp{
			Code:    xerr.ErrInvalidParams.Code(),
			Message: xerr.ErrInvalidParams.Error(),
		}, nil
	}

	claim, err := ParseJWT(auth.Token)
	if err != nil {
		log.Warn(ctx, "parse jwt token failed", log.String("err", err.Error()))
		return &sessionpb.LoginResp{
			Code:    xerr.ErrUnauthorized.Code(),
			Message: xerr.ErrUnauthorized.Error(),
		}, nil
	}
	now := time.Now().Unix()
	if now >= claim.ExpireTime {
		log.Warn(ctx, "token is expired", log.String("user_id", claim.UserID), log.Int64("expire_time", claim.ExpireTime))
		return &sessionpb.LoginResp{
			Code:    xerr.ErrUnauthorized.Code(),
			Message: xerr.ErrUnauthorized.Error(),
		}, nil
	}

	session := &sessionpb.Session{
		UserId:       claim.UserID,
		DeviceId:     auth.GetDeviceId(),
		DeviceType:   auth.GetDeviceType(),
		GatewayId:    req.GetGatewayId(),
		ConnId:       req.GetConnId(),
		RemoteAddr:   req.GetRemoteAddr(),
		Status:       sessionpb.SessionStatus_SESSION_STATUS_ONLINE,
		LoginAt:      now,
		LastActiveAt: now,
		ExpireAt:     claim.ExpireTime,
		Meta:         auth.GetMeta(),
	}
	err = s.redis.StoreSession(ctx, session)
	if err != nil {
		log.Error(ctx, "store session failed",
			log.String("err", err.Error()),
			log.String("data", xjson.MarshalString(session)),
		)
		return &sessionpb.LoginResp{
			Code:    xerr.ErrInternalServer.Code(),
			Message: xerr.ErrInternalServer.Error(),
		}, nil
	}

	return &sessionpb.LoginResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
		Data: &sessionpb.LoginData{
			Session: session,
		},
	}, nil
}

// GetSessions 获取用户会话列表
func (s *SessionService) GetSessions(ctx context.Context, req *sessionpb.GetSessionsReq) (*sessionpb.GetSessionsResp, error) {
	var sessions []*sessionpb.Session
	var err error

	if req.DeviceId != "" {
		// 获取指定设备的会话
		session, err := s.redis.GetSession(ctx, req.UserId, req.DeviceId)
		if err != nil {
			if errors.Is(err, redis.ErrSessionNotFound) {
				return &sessionpb.GetSessionsResp{
					Code:    xerr.OK.Code(),
					Message: xerr.OK.Error(),
					Data: &sessionpb.GetSessionsData{
						Sessions: []*sessionpb.Session{},
					},
				}, nil
			}
			log.Error(ctx, "get session failed",
				log.String("err", err.Error()),
				log.String("user_id", req.UserId),
				log.String("device_id", req.DeviceId),
			)
			return &sessionpb.GetSessionsResp{
				Code:    xerr.ErrInternalServer.Code(),
				Message: xerr.ErrInternalServer.Error(),
			}, nil
		}
		sessions = []*sessionpb.Session{session}
	} else {
		// 获取用户所有设备的会话
		sessions, err = s.redis.GetSessionsByUserID(ctx, req.UserId)
		if err != nil {
			log.Error(ctx, "get sessions by user id failed",
				log.String("err", err.Error()),
				log.String("user_id", req.UserId),
			)
			return &sessionpb.GetSessionsResp{
				Code:    xerr.ErrInternalServer.Code(),
				Message: xerr.ErrInternalServer.Error(),
			}, nil
		}
	}

	return &sessionpb.GetSessionsResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
		Data: &sessionpb.GetSessionsData{
			Sessions: sessions,
		},
	}, nil
}

// Kick 踢掉用户会话
func (s *SessionService) Kick(ctx context.Context, req *sessionpb.KickReq) (*sessionpb.KickResp, error) {
	var err error
	if req.DeviceId != "" {
		// 踢掉指定设备的会话
		err = s.redis.DeleteSession(ctx, req.UserId, req.DeviceId)
		if err != nil {
			if errors.Is(err, redis.ErrSessionNotFound) {
				// 会话不存在，返回成功（幂等性）
				return &sessionpb.KickResp{
					Code:    xerr.OK.Code(),
					Message: xerr.OK.Error(),
				}, nil
			}
			log.Error(ctx, "delete session failed",
				log.String("err", err.Error()),
				log.String("user_id", req.UserId),
				log.String("device_id", req.DeviceId),
			)
			return &sessionpb.KickResp{
				Code:    xerr.ErrInternalServer.Code(),
				Message: xerr.ErrInternalServer.Error(),
			}, nil
		}

		log.Info(ctx, "session kicked",
			log.String("user_id", req.UserId),
			log.String("device_id", req.DeviceId),
			log.String("reason", req.Reason),
		)
	} else {
		// 踢掉用户所有设备的会话
		err = s.redis.DeleteSessionsByUserID(ctx, req.UserId)
		if err != nil {
			log.Error(ctx, "delete sessions by user id failed",
				log.String("err", err.Error()),
				log.String("user_id", req.UserId),
			)
			return &sessionpb.KickResp{
				Code:    xerr.ErrInternalServer.Code(),
				Message: xerr.ErrInternalServer.Error(),
			}, nil
		}

		log.Info(ctx, "all sessions kicked",
			log.String("user_id", req.UserId),
			log.String("reason", req.Reason),
		)
	}

	// TODO: 通知 Gateway 关闭连接
	// 这里需要调用 Gateway 的 CloseConn 方法
	// 可以通过消息队列或者直接调用 Gateway 服务

	return &sessionpb.KickResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
	}, nil
}

// RefreshSessionTTL 刷新Session TTL
func (s *SessionService) RefreshSessionTTL(ctx context.Context, req *sessionpb.RefreshSessionTTLReq) (*sessionpb.RefreshSessionTTLResp, error) {
	// 使用Lua脚本刷新Session TTL（保证原子性）
	err := s.redis.RefreshSessionTTL(ctx, req.UserId, req.DeviceId, req.LastActiveAt)
	if err != nil {
		if errors.Is(err, redis.ErrSessionNotFound) {
			log.Warn(ctx, "session not found",
				log.String("user_id", req.UserId),
				log.String("device_id", req.DeviceId),
			)
			return &sessionpb.RefreshSessionTTLResp{
				Code:    xerr.ErrNotFound.Code(),
				Message: xerr.ErrNotFound.Error(),
			}, nil
		}

		log.Error(ctx, "refresh session TTL failed",
			log.String("err", err.Error()),
			log.String("user_id", req.UserId),
			log.String("device_id", req.DeviceId),
		)
		return &sessionpb.RefreshSessionTTLResp{
			Code:    xerr.ErrInternalServer.Code(),
			Message: xerr.ErrInternalServer.Error(),
		}, nil
	}

	log.Debug(ctx, "session TTL refreshed",
		log.String("user_id", req.UserId),
		log.String("device_id", req.DeviceId),
		log.Int64("last_active_at", req.LastActiveAt),
	)

	return &sessionpb.RefreshSessionTTLResp{
		Code:    xerr.OK.Code(),
		Message: xerr.OK.Error(),
	}, nil
}
