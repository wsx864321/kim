package redis

import "github.com/redis/go-redis/v9"

type Instance struct {
	redis *redis.Client
}

func NewInstance(redis *redis.Client) *Instance {
	return &Instance{redis: redis}
}
