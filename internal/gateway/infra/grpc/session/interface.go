package session

import (
	"context"
	sessionpb "github.com/wsx864321/kim/idl/session"
)

// ClientInterface ...
type ClientInterface interface {
	// Login 用户登录
	Login(ctx context.Context, in *sessionpb.LoginReq) (*sessionpb.LoginResp, error)
	// DelSession 删除用户会话
	DelSession(ctx context.Context, in *sessionpb.DelSessionReq) (*sessionpb.DelSessionResp, error)
	// GetSessions 获取用户会话列表
	GetSessions(ctx context.Context, in *sessionpb.GetSessionsReq) (*sessionpb.GetSessionsResp, error)
	// RefreshSessionTTL 刷新会话 TTL
	RefreshSessionTTL(ctx context.Context, in *sessionpb.RefreshSessionTTLReq) (*sessionpb.RefreshSessionTTLResp, error)
}
