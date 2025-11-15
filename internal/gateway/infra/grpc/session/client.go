package session

import (
	"context"
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/pkg/krpc"
	"github.com/wsx864321/kim/pkg/log"
)

// Client Session client
type Client struct {
	cli sessionpb.SessionServiceClient
}

// NewClient 创建 Session 客户端
func NewClient() *Client {
	cli, err := krpc.NewKClient(krpc.WithClientServiceName("kim-session"))
	if err != nil {
		log.Error(nil, "create session client failed",
			log.String("error", err.Error()),
		)
		panic(err)
	}

	return &Client{cli: sessionpb.NewSessionServiceClient(cli.Conn())}
}

// Login 用户登录
func (c *Client) Login(ctx context.Context, in *sessionpb.LoginReq) (*sessionpb.LoginResp, error) {
	return c.cli.Login(ctx, in)
}

// DelSession 删除用户会话
func (c *Client) DelSession(ctx context.Context, in *sessionpb.DelSessionReq) (*sessionpb.DelSessionResp, error) {
	return c.cli.DelSession(ctx, in)
}

// GetSessions 获取用户会话列表
func (c *Client) GetSessions(ctx context.Context, in *sessionpb.GetSessionsReq) (*sessionpb.GetSessionsResp, error) {
	return c.cli.GetSessions(ctx, in)
}

// RefreshSessionTTL 刷新会话 TTL
func (c *Client) RefreshSessionTTL(ctx context.Context, in *sessionpb.RefreshSessionTTLReq) (*sessionpb.RefreshSessionTTLResp, error) {
	return c.cli.RefreshSessionTTL(ctx, in)
}
