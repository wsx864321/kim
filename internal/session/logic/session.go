package logic

import (
	"context"
	"errors"
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
func (s *SessionService) Login(ctx context.Context, auth *sessionpb.AuthInfo, req *sessionpb.LoginReq) (*sessionpb.LoginData, *xerr.Error) {
	claim, err := ParseJWT(auth.Token)
	if err != nil {
		log.Warn(ctx, "parse jwt token failed", log.String("error", err.Error()))
		return nil, xerr.ErrInvalidParams.WithMessage("invalid token")
	}
	now := time.Now().Unix()
	if now >= claim.ExpireTime {
		log.Warn(ctx, "token is expired", log.String("user_id", claim.UserID), log.Int64("expire_time", claim.ExpireTime))
		return nil, xerr.ErrInvalidParams.WithMessage("token is expired")
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
		return nil, xerr.ErrInternalServer
	}

	return &sessionpb.LoginData{
		Session: session,
	}, nil
}

// GetSessions 获取用户会话列表
func (s *SessionService) GetSessions(ctx context.Context, req *sessionpb.GetSessionsReq) (*sessionpb.GetSessionsData, *xerr.Error) {
	// 验证参数
	if req.UserId == "" {
		log.Warn(ctx, "user_id is required")
		return nil, xerr.ErrInvalidParams.WithMessage("user_id is required")
	}

	deviceIDs := req.GetDeviceId()

	// 如果没有指定 device_id，获取该用户所有设备的会话
	if len(deviceIDs) == 0 {
		sessions, err := s.redis.GetSessionsByUserID(ctx, req.UserId)
		if err != nil {
			log.Error(ctx, "get sessions by user id failed",
				log.String("err", err.Error()),
				log.String("user_id", req.UserId),
			)
			return nil, xerr.ErrInternalServer
		}
		return &sessionpb.GetSessionsData{
			Sessions: sessions,
		}, nil
	}

	// 获取指定设备的会话
	sessions := make([]*sessionpb.Session, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		if deviceID == "" {
			log.Warn(ctx, "empty device_id in list, skipping",
				log.String("user_id", req.UserId),
			)
			continue
		}

		session, err := s.redis.GetSession(ctx, req.UserId, deviceID)
		if err != nil {
			if errors.Is(err, redis.ErrSessionNotFound) {
				// 会话不存在，跳过（不返回错误，允许部分设备不存在）
				log.Debug(ctx, "session not found",
					log.String("user_id", req.UserId),
					log.String("device_id", deviceID),
				)
				continue
			}
			log.Error(ctx, "get session failed",
				log.String("err", err.Error()),
				log.String("user_id", req.UserId),
				log.String("device_id", deviceID),
			)
			// 如果获取失败，仍然返回已获取到的会话（部分成功）
			// 但记录错误日志
			continue
		}
		sessions = append(sessions, session)
	}

	return &sessionpb.GetSessionsData{
		Sessions: sessions,
	}, nil
}

// Kick 踢掉用户会话
func (s *SessionService) Kick(ctx context.Context, req *sessionpb.KickReq) *xerr.Error {
	var err error
	if req.DeviceId != "" {
		// 踢掉指定设备的会话
		err = s.redis.DeleteSession(ctx, req.UserId, req.DeviceId)
		if err != nil {
			if errors.Is(err, redis.ErrSessionNotFound) {
				// 会话不存在，返回成功（幂等性）
				return nil
			}
			log.Error(ctx, "delete session failed",
				log.String("err", err.Error()),
				log.String("user_id", req.UserId),
				log.String("device_id", req.DeviceId),
			)
			return xerr.ErrInternalServer
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
			return xerr.ErrInternalServer
		}

		log.Info(ctx, "all sessions kicked",
			log.String("user_id", req.UserId),
			log.String("reason", req.Reason),
		)
	}

	// TODO: 通知 Gateway 关闭连接
	// 这里需要调用 Gateway 的 CloseConn 方法
	// 可以通过消息队列或者直接调用 Gateway 服务

	return nil
}

// RefreshSessionTTL 刷新Session TTL
func (s *SessionService) RefreshSessionTTL(ctx context.Context, req *sessionpb.RefreshSessionTTLReq) *xerr.Error {
	// 使用Lua脚本刷新Session TTL（保证原子性）
	err := s.redis.RefreshSessionTTL(ctx, req.UserId, req.DeviceId, req.LastActiveAt)
	if err != nil {
		if errors.Is(err, redis.ErrSessionNotFound) {
			log.Warn(ctx, "session not found",
				log.String("user_id", req.UserId),
				log.String("device_id", req.DeviceId),
			)
			return xerr.ErrSessionNotFound
		}

		log.Error(ctx, "refresh session TTL failed",
			log.String("err", err.Error()),
			log.String("user_id", req.UserId),
			log.String("device_id", req.DeviceId),
		)
		return xerr.ErrInternalServer
	}

	log.Debug(ctx, "session TTL refreshed",
		log.String("user_id", req.UserId),
		log.String("device_id", req.DeviceId),
		log.Int64("last_active_at", req.LastActiveAt),
	)

	return nil
}

// DelSession 删除用户会话
func (s *SessionService) DelSession(ctx context.Context, req *sessionpb.DelSessionReq) *xerr.Error {
	deviceIDs := req.GetDeviceId()

	// 如果没有指定 device_id，删除该用户所有会话
	if len(deviceIDs) == 0 {
		err := s.redis.DeleteSessionsByUserID(ctx, req.UserId)
		if err != nil {
			log.Error(ctx, "delete sessions by user id failed",
				log.String("err", err.Error()),
				log.String("user_id", req.UserId),
			)
			return xerr.ErrInternalServer
		}

		log.Info(ctx, "all sessions deleted",
			log.String("user_id", req.UserId),
			log.String("reason", req.Reason),
		)
		return nil
	}

	// 删除指定设备的会话
	var deletedCount int
	var lastErr error
	for _, deviceID := range deviceIDs {
		if deviceID == "" {
			log.Warn(ctx, "empty device_id in list, skipping",
				log.String("user_id", req.UserId),
			)
			continue
		}

		err := s.redis.DeleteSession(ctx, req.UserId, deviceID)
		if err != nil {
			if errors.Is(err, redis.ErrSessionNotFound) {
				// 会话不存在，记录日志但继续处理（幂等性）
				log.Debug(ctx, "session not found, already deleted",
					log.String("user_id", req.UserId),
					log.String("device_id", deviceID),
				)
				deletedCount++ // 视为成功（幂等性）
				continue
			}
			log.Error(ctx, "delete session failed",
				log.String("err", err.Error()),
				log.String("user_id", req.UserId),
				log.String("device_id", deviceID),
			)
			lastErr = err
			continue
		}

		deletedCount++
		log.Info(ctx, "session deleted",
			log.String("user_id", req.UserId),
			log.String("device_id", deviceID),
			log.String("reason", req.Reason),
		)
	}

	// 如果所有删除都失败，返回错误
	if deletedCount == 0 && lastErr != nil {
		return xerr.ErrInternalServer.WithMessage(lastErr.Error())
	}

	// 至少部分成功
	if deletedCount < len(deviceIDs) && lastErr != nil {
		log.Warn(ctx, "some sessions deletion failed",
			log.String("user_id", req.UserId),
			log.Int("total", len(deviceIDs)),
			log.Int("deleted", deletedCount),
			log.String("last_error", lastErr.Error()),
		)
		// 仍然返回成功，因为至少部分删除成功
	}

	return nil
}
