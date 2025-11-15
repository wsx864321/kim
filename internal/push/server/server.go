package server

import (
	"context"
	pushpb "github.com/wsx864321/kim/idl/push"
	"github.com/wsx864321/kim/internal/gateway/infra/grpc/session"
	"github.com/wsx864321/kim/internal/push/handler"
	"github.com/wsx864321/kim/internal/push/infra/grpc/gateway"
	"github.com/wsx864321/kim/internal/push/logic"
	"github.com/wsx864321/kim/internal/push/pkg/config"
	"github.com/wsx864321/kim/pkg/krpc"
	"github.com/wsx864321/kim/pkg/krpc/registry"
	"github.com/wsx864321/kim/pkg/krpc/registry/etcd"
	"github.com/wsx864321/kim/pkg/log"
	"google.golang.org/grpc"
)

// Run 启动 Push 服务端
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

	// 创建 Push Handler
	pushHandler := createPushHandler()

	// 创建 gRPC 服务器
	grpcServer := krpc.NewPServer(
		krpc.WithServiceName(config.GetPushServiceName()),
		krpc.WithPort(config.GetPushServicePort()),
		krpc.WithRegistry(createEtcdRegistry()),
	)

	// 注册 Push gRPC 服务
	grpcServer.RegisterService(func(server *grpc.Server) {
		pushpb.RegisterPushServiceServer(server, pushHandler)
	})

	log.Info(ctx, "push server starting",
		log.String("service_name", config.GetPushServiceName()),
		log.Int("port", config.GetPushServicePort()),
	)

	// 启动 gRPC 服务（会阻塞）
	grpcServer.Start(ctx)
}

// createPushHandler 创建 PushHandler 实例
func createPushHandler() *handler.PushHandler {
	// 创建 Push Service
	pushService := logic.NewPushService(
		session.NewClient(),
		createGatewayManager(),
	)

	return handler.NewPushHandler(pushService)
}

// createGatewayManager 创建 Gateway 客户端管理器
func createGatewayManager() *gateway.ClientManager {
	return gateway.NewClientManager()
}

// createEtcdRegistry 创建 Etcd 注册中心
func createEtcdRegistry() registry.Registrar {
	r, err := etcd.NewETCDRegister(etcd.WithEndpoints(config.GetRegistryEndpoints()))
	if err != nil {
		panic(err)
	}

	return r
}
