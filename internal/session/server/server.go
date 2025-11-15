package server

import (
	"context"
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/internal/session/handler"
	"github.com/wsx864321/kim/internal/session/infra/redis"
	"github.com/wsx864321/kim/internal/session/logic"
	"github.com/wsx864321/kim/internal/session/pkg/config"
	"github.com/wsx864321/kim/pkg/krpc"
	"github.com/wsx864321/kim/pkg/krpc/registry"
	"github.com/wsx864321/kim/pkg/krpc/registry/etcd"
	"github.com/wsx864321/kim/pkg/log"
	"google.golang.org/grpc"
)

// Run 启动 Session 服务端
func Run(configPath string) {
	// 初始化配置
	config.Init(configPath)

	// 初始化日志
	log.InitLogger(
		log.WithDebug(config.GetLogDebug()),
		log.WithLogDir(config.GetLogDir()),
		log.WithHistoryLogFileName(config.GetLogFilename()),
	)

	s := krpc.NewPServer(
		krpc.WithServiceName(config.GetSessionServiceName()),
		krpc.WithPort(config.GetSessionServicePort()),
		krpc.WithRegistry(createEtcdRegistry()),
	)

	// 注册 Session 服务
	s.RegisterService(func(server *grpc.Server) {
		sessionpb.RegisterSessionServiceServer(server, createSessionHandler())
	})

	// 启动服务（会阻塞）
	s.Start(context.Background())
}

// createSessionHandler 创建 Session 控制器
func createSessionHandler() *handler.SessionHandler {
	return handler.NewSessionHandler(
		logic.NewSessionService(
			redis.NewInstance(),
		),
	)
}

// createEtcdRegistry 创建 Etcd 注册中心
func createEtcdRegistry() registry.Registrar {
	r, err := etcd.NewETCDRegister(etcd.WithEndpoints(config.GetRegistryEndpoints()))
	if err != nil {
		panic(err)
	}

	return r
}
