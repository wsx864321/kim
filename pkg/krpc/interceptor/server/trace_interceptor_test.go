package server

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ktrace "github.com/wsx864321/kim/pkg/krpc/trace"
	"google.golang.org/grpc"
)

func TestTraceUnaryServerInterceptor(t *testing.T) {
	ktrace.StartAgent()
	defer ktrace.StopAgent()

	//cc := new(grpc.ClientConn)
	TraceUnaryServerInterceptor()(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/helloworld.Greeter/SayHello",
	}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})

	TraceUnaryServerInterceptor()(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/helloworld.Greeter/SayBye",
	}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.DataLoss, "dummy")
	})
}
