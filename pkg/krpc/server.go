package krpc

import (
	"context"
	"fmt"
	serverinterceptor "github.com/wsx864321/kim/pkg/krpc/interceptor/server"
	"github.com/wsx864321/kim/pkg/krpc/registry"
	"github.com/wsx864321/kim/pkg/log"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ServiceRegisterFunc func(*grpc.Server)

type KServer struct {
	serverOptions
	registers    []ServiceRegisterFunc
	interceptors []grpc.UnaryServerInterceptor
}

func NewPServer(opts ...ServerOption) *KServer {
	opt := serverOptions{}
	for _, o := range opts {
		o(&opt)
	}

	return &KServer{
		opt,
		make([]ServiceRegisterFunc, 0),
		make([]grpc.UnaryServerInterceptor, 0),
	}
}

// RegisterService ...
// eg :
//
//	p.RegisterService(func(server *grpc.Server) {
//	    test.RegisterGreeterServer(server, &Server{})
//	})
func (p *KServer) RegisterService(register ...ServiceRegisterFunc) {
	p.registers = append(p.registers, register...)
}

// RegisterUnaryServerInterceptor 注册自定义拦截器，例如限流拦截器或者自己的一些业务自定义拦截器
func (p *KServer) RegisterUnaryServerInterceptor(i grpc.UnaryServerInterceptor) {
	p.interceptors = append(p.interceptors, i)
}

// Start 开启server
func (p *KServer) Start(ctx context.Context) {
	service := registry.Service{
		Name: p.serviceName,
		Endpoints: []*registry.Endpoint{
			{
				ServerName: p.serviceName,
				IP:         p.ip,
				Port:       p.port,
				Weight:     p.weight,
				Enable:     true,
			},
		},
	}

	// 加载中间件
	interceptors := []grpc.UnaryServerInterceptor{
		serverinterceptor.RecoveryUnaryServerInterceptor(),
		serverinterceptor.TraceUnaryServerInterceptor(),
		serverinterceptor.MetricUnaryServerInterceptor(p.serviceName),
	}
	interceptors = append(interceptors, p.interceptors...)

	s := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))

	// 注册服务
	for _, register := range p.registers {
		register(s)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", p.ip, p.port))
	if err != nil {
		panic(err)
	}

	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
	// 服务注册
	p.registry.Register(ctx, &service)

	log.Info(context.Background(), "start PRCP success")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-c
		switch sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			s.Stop()
			p.registry.UnRegister(ctx, &service)
			time.Sleep(time.Second)
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}

}
