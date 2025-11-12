package krpc

import (
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
	"time"

	"google.golang.org/grpc/resolver"

	clientinterceptor "github.com/wsx864321/kim/pkg/krpc/interceptor/client"
	presolver "github.com/wsx864321/kim/pkg/krpc/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

const (
	dialTimeout = 5 * time.Second
)

type KClient struct {
	clientOptions
	conn *grpc.ClientConn
}

// NewKClient ...
func NewKClient(opts ...ClientOption) (*KClient, error) {
	opt := clientOptions{}
	for _, o := range opts {
		o(&opt)
	}

	p := &KClient{
		clientOptions: opt,
	}

	if p.registry != nil {
		resolver.Register(presolver.NewRegistryBuilder(p.registry))
	}

	conn, err := p.dial()
	p.conn = conn

	return p, err
}

// Conn return *grpc.ClientConn
func (p *KClient) Conn() *grpc.ClientConn {
	return p.conn
}

func (p *KClient) dial() (*grpc.ClientConn, error) {
	svcCfg := fmt.Sprintf(`{"loadBalancingPolicy":"%s"}`, roundrobin.Name)
	balancerOpt := grpc.WithDefaultServiceConfig(svcCfg)

	interceptors := []grpc.UnaryClientInterceptor{
		clientinterceptor.TraceUnaryClientInterceptor(),
		clientinterceptor.MetricUnaryClientInterceptor(),
	}
	interceptors = append(interceptors, p.interceptors...)

	options := []grpc.DialOption{
		balancerOpt,
		grpc.WithChainUnaryInterceptor(interceptors...),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if p.direct {
		return grpc.NewClient(p.url, options...)
	}

	return grpc.NewClient(fmt.Sprintf("discov:///%v", p.serviceName), options...)
}
