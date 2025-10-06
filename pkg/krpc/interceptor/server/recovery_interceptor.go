package server

import (
	"context"
	"github.com/wsx864321/kim/pkg/log"
	"runtime"

	"google.golang.org/grpc"
)

// RecoveryUnaryServerInterceptor recovery中间件最好放在第一个去执行
func RecoveryUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if err := recover(); err != nil {
				stack := make([]byte, 4096)
				runtime.Stack(stack, false)
				log.Error(ctx, "panic recovered", log.String("stack", string(stack)))
			}

		}()

		return handler(ctx, req)
	}
}
