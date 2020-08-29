package grpc

import (
	"gome/gomengine/util"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net"
)

var Conf *util.MeConfig

type gRPC struct {
	Listener net.Listener
	Protocol string
	RPCurl   string
}

func init() {
	confFile, _ := ioutil.ReadFile("config.yaml")
	yaml.Unmarshal(confFile, &Conf)
}

func NewRpcListener() *gRPC {
	host := Conf.GRPCConf.Host
	port := Conf.GRPCConf.Port
	RPCurl := host + ":" + port
	gRPC := &gRPC{Protocol: "tcp", RPCurl: RPCurl}

	var err error
	gRPC.Listener, err = net.Listen(gRPC.Protocol, gRPC.RPCurl)
	if err != nil {
		panic("监听失败")
	} else {
		log.Println("撮合服务正在监听: " + host + ":" + port)
	}

	return gRPC
}
