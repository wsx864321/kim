package server

import (
	"context"
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/internal/session/controller"
	logic "github.com/wsx864321/kim/internal/session/logic"
	"github.com/wsx864321/kim/internal/session/pkg/config"
	"github.com/wsx864321/kim/internal/session/repository/redis"
	"github.com/wsx864321/kim/pkg/krpc"
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
		krpc.WithIP(config.GetSessionServiceIP()),
		krpc.WithPort(config.GetSessionServicePort()),
	)

	// 注册 Session 服务
	s.RegisterService(func(server *grpc.Server) {
		sessionpb.RegisterSessionServiceServer(server, CreateSessionController())
	})

	// 启动服务（会阻塞）
	s.Start(context.Background())
}

// CreateSessionController 创建 Session 控制器
func CreateSessionController() *controller.SessionController {
	return controller.NewSessionController(
		logic.NewSessionService(
			redis.NewInstance(),
		),
	)
}
