package redis

import (
	"github.com/go-redis/redis/v8"
	"gome/gomengine/util"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

func NewRedisClient() *redis.Client {
	conf := &util.MeConfig{}
	yamlFile, _ := ioutil.ReadFile("config.yaml")
	yaml.Unmarshal(yamlFile, conf)
	host := conf.CacheConf.Host
	port := conf.CacheConf.Port
	//password := conf.CacheConf.Password
	cache := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return cache
}
