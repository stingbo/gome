package redis

import (
	"github.com/go-redis/redis/v8"
	"gome/utils"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

var Conf *utils.MeConfig

func init() {
	confFile, _ := ioutil.ReadFile("config.yaml")
	yaml.Unmarshal(confFile, &Conf)
}

func NewRedisClient() *redis.Client {
	host := Conf.CacheConf.Host
	port := Conf.CacheConf.Port
	//password := conf.CacheConf.Password
	cache := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return cache
}
