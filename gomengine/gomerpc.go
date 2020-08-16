package main

import (
	"log"
	"net"
	"github.com/unknwon/goconfig"
)

func getRpcListener() net.Listener {
	config, err := goconfig.LoadConfigFile("config.ini")
	host,_ := config.GetValue("grpc", "host")
	port,_ := config.GetValue("grpc", "port")
	//log.Println(err)
	//log.Println("-------"+host)
	//log.Println(port)
	if err != nil {
		panic("配置读取失败")
	}

	listener, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		panic("监听失败")
	} else {
		log.Println("撮合服务正在监听: " + host + ":" + port)
	}

	return listener
}
