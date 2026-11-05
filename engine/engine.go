package engine

import (
	"context"

	goredis "github.com/go-redis/redis/v8"
	"gome/redis"
	"gome/utils"
)

const (
	_ = iota
	ADD
	DEL
	timeFormat = "2006-01-02 15:04:05"
)

var (
	ctx   = context.Background()
	cache *goredis.Client
	Conf  = &utils.MeConfig{}
)

type MatchResult struct {
	EventID     string
	Action      int8
	Node        OrderNode
	MatchNode   OrderNode
	MatchVolume float64
	MatchTime   string
}

func Configure(config *utils.MeConfig) {
	Conf = config
	cache = redis.NewRedisClient(config.CacheConf)
}
