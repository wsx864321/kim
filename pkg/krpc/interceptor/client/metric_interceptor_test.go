package client

import (
	"context"
	"google.golang.org/grpc"
	"testing"
	"time"

	"github.com/wsx864321/kim/pkg/krpc/prome"
)

func TestMetricUnaryClientInterceptor(t *testing.T) {
	prome.StartAgent("0.0.0.0", 8927)

	cc := new(grpc.ClientConn)
	for {
		MetricUnaryClientInterceptor()(context.TODO(), "/create", nil, nil, cc,
			func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
				opts ...grpc.CallOption) error {
				time.Sleep(20 * time.Millisecond)
				return nil
			})

	}

}
