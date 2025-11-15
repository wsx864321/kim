package server

import (
	"context"
	"github.com/wsx864321/kim/internal/gateway/event"
	"github.com/wsx864321/kim/internal/gateway/infra/grpc/session"
	"time"

	gatewaypb "github.com/wsx864321/kim/idl/gateway"
	"github.com/wsx864321/kim/internal/gateway/conn"
	"github.com/wsx864321/kim/internal/gateway/handler"
	"github.com/wsx864321/kim/internal/gateway/pkg/config"
	"github.com/wsx864321/kim/pkg/krpc"
	"github.com/wsx864321/kim/pkg/log"
	"google.golang.org/grpc"
)

// Run 启动 Gateway 服务端
func Run(configPath string) {
	// 初始化配置
	config.Init(configPath)

	// 初始化日志
	log.InitLogger(
		log.WithDebug(config.GetLogDebug()),
		log.WithLogDir(config.GetLogDir()),
		log.WithHistoryLogFileName(config.GetLogFilename()),
	)

	ctx := context.Background()

	// 创建TCP Transport
	tcpTransport, err := createTCPTransport()
	if err != nil {
		log.Error(ctx, "create tcp transport failed", log.String("error", err.Error()))
		panic(err)
	}

	// 创建Session客户端管理器
	sessionClient := session.NewClient()

	// 创建Handler
	gatewayHandler := handler.NewGatewayHandler(sessionClient, tcpTransport)

	// 设置Handler到Transport
	tcpTransport.SetHandler(event.NewEvent(sessionClient))

	// 启动TCP Transport
	if err := tcpTransport.Start(); err != nil {
		log.Error(ctx, "start tcp transport failed", log.String("error", err.Error()))
		panic(err)
	}

	log.Info(ctx, "tcp transport started", log.Int("port", config.GetGatewayTCPPort()))

	// 创建gRPC服务器
	grpcServer := krpc.NewPServer(
		krpc.WithServiceName(config.GetGatewayServiceName()),
		krpc.WithPort(config.GetGatewayServicePort()),
	)

	// 注册Gateway gRPC服务
	grpcServer.RegisterService(func(server *grpc.Server) {
		gatewaypb.RegisterGatewayServiceServer(server, gatewayHandler)
	})

	log.Info(ctx, "gateway server starting",
		log.String("service_name", config.GetGatewayServiceName()),
		log.Int("grpc_port", config.GetGatewayServicePort()),
		log.Int("tcp_port", config.GetGatewayTCPPort()),
		log.String("gateway_id", config.GetGatewayID()),
	)

	// 启动gRPC服务（会阻塞）
	grpcServer.Start(ctx)
}

// createTCPTransport 创建TCP Transport
func createTCPTransport() (conn.Transport, error) {
	tcpPort := config.GetGatewayTCPPort()
	gatewayID := config.GetGatewayID()
	heartbeatTimeout := time.Duration(config.GetHeartbeatTimeout()) * time.Second
	refreshTTLInterval := time.Duration(config.GetRefreshTTLInterval()) * time.Second

	opts := []conn.TCPOption{
		conn.WithGatewayID(gatewayID),
		conn.WithTCPHeartbeatTimeout(heartbeatTimeout),
		conn.WithRefreshTTLInterval(refreshTTLInterval),
	}

	// 设置工作协程数量
	if numWorkers := config.GetNumWorkers(); numWorkers > 0 {
		opts = append(opts, conn.WithTCPNumWorkers(numWorkers))
	}

	return conn.NewTCPTransport(tcpPort, opts...)
}
