package redis

import (
	"net"

	"github.com/go-redis/redis/v8"
	"gome/utils"
)

func NewRedisClient(config utils.Cache) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(config.Host, config.Port),
		Password: config.Password,
	})
}
