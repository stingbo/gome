package Redis

import (
	"github.com/go-redis/redis/v8"
	"github.com/unknwon/goconfig"
)

func NewRedisClient() *redis.Client {
	config, err := goconfig.LoadConfigFile("config.ini")
	if err != nil {
		panic("配置读取失败")
	}
	host, _ := config.GetValue("redis", "host")
	port, _ := config.GetValue("redis", "port")
	//password,_ := config.GetValue("redis", "password")
	gomerdb := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return gomerdb
}
