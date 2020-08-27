package grpc

import (
	"gome/gomengine/util"
	"gopkg.in/yaml.v2"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net"
)

type gRPC struct {
	Listener net.Listener
	Protocol string
	RPCurl   string
}

func NewRpcListener() *gRPC {
	conf := &util.MeConfig{}
	yamlFile, err := ioutil.ReadFile("config.yaml")
	yaml.Unmarshal(yamlFile, conf)
	host := conf.GRPCConf.Host
	port := conf.GRPCConf.Port
	RPCurl := host + ":" + port
	gRPC := &gRPC{Protocol: "tcp", RPCurl: RPCurl}

	gRPC.Listener, err = net.Listen(gRPC.Protocol, gRPC.RPCurl)
	if err != nil {
		panic("监听失败")
	} else {
		log.Println("撮合服务正在监听: " + host + ":" + port)
	}

	return gRPC
}
