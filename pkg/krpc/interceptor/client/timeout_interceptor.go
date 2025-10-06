package client

import (
	"context"
	"github.com/wsx864321/kim/pkg/log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// TimeoutUnaryClientInterceptor ...
func TimeoutUnaryClientInterceptor(timeout time.Duration, slowThreshold time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		now := time.Now()
		// 若无自定义超时设置，默认设置超时
		_, ok := ctx.Deadline()
		if !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		var p peer.Peer

		err := invoker(ctx, method, req, reply, cc, append(opts, grpc.Peer(&p))...)

		du := time.Since(now)
		remoteIP := ""
		if p.Addr != nil {
			remoteIP = p.Addr.String()
		}

		if slowThreshold > time.Duration(0) && du > slowThreshold {
			//log.CtxErrorf(ctx, "grpc slowlog:method%s,tagert:%s,cost:%v,remotIP:%s", method, cc.Target(), du, remoteIP)
			log.Warn(ctx,
				"grpc slow log",
				log.String("method", method),
				log.String("target", cc.Target()),
				log.Duration("cost", du),
				log.String("remoteIP", remoteIP),
			)

		}
		return err
	}
}
