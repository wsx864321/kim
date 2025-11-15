package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/pkg/log"
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
	sessionExpire = 200 * time.Second
)

// StoreSession 存储Session（使用Lua脚本保证原子性）
func (i *Instance) StoreSession(ctx context.Context, session *sessionpb.Session) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session failed: %w", err)
	}

	sessionKey := buildUserSessionKey(session.GetUserId(), session.GetDeviceId())
	setKey := buildUserSessionsSetKey(session.GetUserId())
	expireSeconds := int64(sessionExpire.Seconds())

	// 使用Lua脚本原子性地存储session和添加到集合
	_, err = i.storeSessionLuaScript.Run(ctx, i.redis, []string{sessionKey, setKey},
		string(raw),
		session.GetDeviceId(),
		fmt.Sprintf("%d", expireSeconds),
	).Result()
	if err != nil {
		return fmt.Errorf("store session failed: %w", err)
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

// GetSessionsByUserID 获取用户所有会话（使用Lua脚本保证原子性）
func (i *Instance) GetSessionsByUserID(ctx context.Context, userID string) ([]*sessionpb.Session, error) {
	setKey := buildUserSessionsSetKey(userID)

	// 使用Lua脚本原子性地获取所有会话并清理过期数据
	result, err := i.getSessionsByUserIDLuaScript.Run(ctx, i.redis, []string{setKey}, userID).Result()
	if err != nil {
		return nil, fmt.Errorf("get sessions by user id failed: %w", err)
	}

	// 解析结果
	resultSlice, ok := result.([]interface{})
	if !ok {
		return []*sessionpb.Session{}, nil
	}

	if len(resultSlice) == 0 {
		return []*sessionpb.Session{}, nil
	}

	// 反序列化所有session
	sessions := make([]*sessionpb.Session, 0, len(resultSlice))
	for _, item := range resultSlice {
		sessionData, ok := item.(string)
		if !ok {
			continue
		}

		var session sessionpb.Session
		if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
			log.Warn(ctx, "unmarshal session failed",
				log.String("user_id", userID),
				log.String("error", err.Error()),
			)
			continue
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// DeleteSession 删除会话（使用Lua脚本保证原子性）
func (i *Instance) DeleteSession(ctx context.Context, userID, deviceID string) error {
	sessionKey := buildUserSessionKey(userID, deviceID)
	setKey := buildUserSessionsSetKey(userID)

	// 使用Lua脚本原子性地删除session和从集合移除
	result, err := i.deleteSessionLuaScript.Run(ctx, i.redis, []string{sessionKey, setKey}, deviceID).Result()
	if err != nil {
		return fmt.Errorf("delete session failed: %w", err)
	}

	// 检查结果（1表示成功，0表示session不存在）
	resultInt := result.(int64)
	if resultInt == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteSessionsByUserID 删除用户所有会话（使用Lua脚本保证原子性）
func (i *Instance) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	setKey := buildUserSessionsSetKey(userID)

	// 使用Lua脚本原子性地删除所有session和集合
	_, err := i.deleteSessionsByUserIDLuaScript.Run(ctx, i.redis, []string{setKey}, userID).Result()
	if err != nil {
		return fmt.Errorf("delete sessions by user id failed: %w", err)
	}

	return nil
}

// RefreshSessionTTL 刷新Session TTL（使用Lua脚本保证原子性）
func (i *Instance) RefreshSessionTTL(ctx context.Context, userID, deviceID string, lastActiveAt int64) error {
	sessionKey := buildUserSessionKey(userID, deviceID)
	expireSeconds := int64(sessionExpire.Seconds())

	// 使用Lua脚本保证原子性操作
	// 参数需要转换为字符串
	result, err := i.refreshSessionTTLLuaScript.Run(ctx, i.redis, []string{sessionKey}, fmt.Sprintf("%d", lastActiveAt), fmt.Sprintf("%d", expireSeconds)).Result()
	if err != nil {
		return fmt.Errorf("refresh session TTL failed: %w", err)
	}

	// 检查结果（1表示成功，0表示session不存在，-1表示JSON解析失败）
	resultInt := result.(int64)
	if resultInt == 0 {
		return ErrSessionNotFound
	}
	if resultInt == -1 {
		return fmt.Errorf("failed to parse session JSON")
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
