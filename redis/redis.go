package redis

import (
	"net"

	"github.com/go-redis/redis/v8"
	"github.com/stingbo/gome/utils"
)

func NewRedisClient(config utils.Cache) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:       net.JoinHostPort(config.Host, config.Port),
		Password:   config.Password,
		MaxRetries: -1, // A failed EXEC may already have committed. Replay the order instead.
	})
}
