package redis

import (
	"context"
	sessionpb "github.com/wsx864321/kim/idl/session"
)

type InstanceInterface interface {
	// StoreSession 存储Session
	StoreSession(ctx context.Context, session *sessionpb.Session) error
	// GetSession 获取单个会话（根据 userID 和 deviceID）
	GetSession(ctx context.Context, userID, deviceID string) (*sessionpb.Session, error)
	// GetSessionsByUserID 获取用户所有会话
	GetSessionsByUserID(ctx context.Context, userID string) ([]*sessionpb.Session, error)
	// DeleteSession 删除会话
	DeleteSession(ctx context.Context, userID, deviceID string) error
	// DeleteSessionsByUserID 删除用户所有会话
	DeleteSessionsByUserID(ctx context.Context, userID string) error
	// RefreshSessionTTL 刷新Session TTL（使用Lua脚本保证原子性）
	RefreshSessionTTL(ctx context.Context, userID, deviceID string, lastActiveAt int64) error
}
