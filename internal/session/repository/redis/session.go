package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/pkg/log"
	"time"
)

var (
	ErrSessionNotFound = errors.New("session not found")
)

const (
	// userSessionKey 单个会话 Key 格式: kim:user:session:{user_id}:{device_id}
	userSessionKey = "kim:user:session:{%s}:%s"
	// userSessionsSetKey 用户会话集合 Key 格式: kim:user:sessions:{user_id}
	// 用于存储用户的所有 device_id，方便快速查询
	userSessionsSetKey = "kim:user:sessions:{%s}"

	// sessionExpire 会话过期时间，7 天
	sessionExpire = 7 * 24 * time.Hour
)

// StoreSession 存储Session
func (i *Instance) StoreSession(ctx context.Context, session *sessionpb.Session) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session failed: %w", err)
	}

	// 存储会话数据
	sessionKey := buildUserSessionKey(session.GetUserId(), session.GetDeviceId())
	if err := i.redis.Set(ctx, sessionKey, string(raw), sessionExpire).Err(); err != nil {
		return fmt.Errorf("set session failed: %w", err)
	}

	// 将 device_id 添加到用户会话集合中
	setKey := buildUserSessionsSetKey(session.GetUserId())
	if err := i.redis.SAdd(ctx, setKey, session.GetDeviceId()).Err(); err != nil {
		return fmt.Errorf("add device to sessions set failed: %w", err)
	}

	// 设置集合过期时间
	if err := i.redis.Expire(ctx, setKey, sessionExpire).Err(); err != nil {
		return fmt.Errorf("expire sessions set failed: %w", err)
	}

	return nil
}

// GetSession 获取单个会话（根据 userID 和 deviceID）
func (i *Instance) GetSession(ctx context.Context, userID, deviceID string) (*sessionpb.Session, error) {
	sessionKey := buildUserSessionKey(userID, deviceID)
	val, err := i.redis.Get(ctx, sessionKey).Result()
	if err != nil {
		// 检查是否是 key 不存在的错误
		if errors.Is(err, redis.Nil) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session failed: %w", err)
	}

	var session sessionpb.Session
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, fmt.Errorf("unmarshal session failed: %w", err)
	}

	return &session, nil
}

// GetSessionsByUserID 获取用户所有会话
func (i *Instance) GetSessionsByUserID(ctx context.Context, userID string) ([]*sessionpb.Session, error) {
	// 从集合中获取所有 device_id
	setKey := buildUserSessionsSetKey(userID)
	deviceIDs, err := i.redis.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil, fmt.Errorf("get device ids failed: %w", err)
	}

	if len(deviceIDs) == 0 {
		return []*sessionpb.Session{}, nil
	}

	// 批量获取所有会话
	sessions := make([]*sessionpb.Session, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		session, err := i.GetSession(ctx, userID, deviceID)
		if err != nil {
			if errors.Is(err, ErrSessionNotFound) {
				// 会话已过期，从集合中移除
				i.redis.SRem(ctx, setKey, deviceID)
				continue
			}
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// DeleteSession 删除会话
func (i *Instance) DeleteSession(ctx context.Context, userID, deviceID string) error {
	sessionKey := buildUserSessionKey(userID, deviceID)

	// 删除会话数据
	if err := i.redis.Del(ctx, sessionKey).Err(); err != nil {
		return fmt.Errorf("delete session failed: %w", err)
	}

	// 从用户会话集合中移除 device_id
	setKey := buildUserSessionsSetKey(userID)
	if err := i.redis.SRem(ctx, setKey, deviceID).Err(); err != nil {
		return fmt.Errorf("remove device from sessions set failed: %w", err)
	}

	return nil
}

// DeleteSessionsByUserID 删除用户所有会话
func (i *Instance) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	// 获取所有 device_id
	setKey := buildUserSessionsSetKey(userID)
	deviceIDs, err := i.redis.SMembers(ctx, setKey).Result()
	if err != nil {
		return fmt.Errorf("get device ids failed: %w", err)
	}

	// 删除所有会话
	for _, deviceID := range deviceIDs {
		if err := i.DeleteSession(ctx, userID, deviceID); err != nil {
			log.Error(ctx,
				"delete session failed",
				log.Any("user", userID),
				log.Any("device", deviceID),
				log.String("err", err.Error()),
			)
			continue
		}
	}

	// 删除集合
	if err := i.redis.Del(ctx, setKey).Err(); err != nil {
		return fmt.Errorf("delete sessions set failed: %w", err)
	}

	return nil
}

// buildUserSessionKey 构建用户会话 Key
func buildUserSessionKey(userID, deviceID string) string {
	return fmt.Sprintf(userSessionKey, userID, deviceID)
}

// buildUserSessionsSetKey 构建用户会话集合 Key
func buildUserSessionsSetKey(userID string) string {
	return fmt.Sprintf(userSessionsSetKey, userID)
}
