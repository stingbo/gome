package grpc

import (
	"github.com/unknwon/goconfig"
	"log"
	"net"
)

type gRPC struct {
	Listener net.Listener
	Protocol string
	RPCurl   string
}

func NewRpcListener() *gRPC {
	config, err := goconfig.LoadConfigFile("config.ini")
	host, _ := config.GetValue("grpc", "host")
	port, _ := config.GetValue("grpc", "port")
	if err != nil {
		panic("配置读取失败")
	}
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
