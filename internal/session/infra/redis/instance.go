package redis

import (
	"github.com/redis/go-redis/v9"
	"github.com/wsx864321/kim/internal/session/pkg/config"
)

type Instance struct {
	redis                          *redis.Client
	refreshSessionTTLLuaScript     *redis.Script
	storeSessionLuaScript          *redis.Script
	getSessionsByUserIDLuaScript   *redis.Script
	deleteSessionLuaScript         *redis.Script
	deleteSessionsByUserIDLuaScript *redis.Script
}

// NewInstance 创建 Redis 实例
func NewInstance() *Instance {
	// todo 当前都是非集群模式，后续可支持集群模式
	endpoint := config.GetSessionServiceRedisEndpoint()
	if endpoint == "" {
		panic("session.redis.endpoint is required")
	}

	//password := config.GetSessionServiceRedisPassword()
	//db := config.GetSessionServiceRedisDB()
	poolSize := config.GetSessionServiceRedisPoolSize()
	minIdleConns := config.GetSessionServiceRedisMinIdleConns()

	cli := redis.NewClient(&redis.Options{
		Addr: endpoint,
		//Password:     password,
		//DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
	})
	return &Instance{
		redis:                          cli,
		refreshSessionTTLLuaScript:     redis.NewScript(refreshSessionTTLLuaScript),
		storeSessionLuaScript:          redis.NewScript(storeSessionLuaScript),
		getSessionsByUserIDLuaScript:   redis.NewScript(getSessionsByUserIDLuaScript),
		deleteSessionLuaScript:         redis.NewScript(deleteSessionLuaScript),
		deleteSessionsByUserIDLuaScript: redis.NewScript(deleteSessionsByUserIDLuaScript),
	}
}
