package main

import (
	"fmt"
	"gome/api"
	"gome/engine"
	rpc "gome/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	rpcListener := rpc.NewRpcListener()
	listener := rpcListener.Listener

	rpcServer := grpc.NewServer()
	api.RegisterOrderServer(rpcServer, &engine.Order{})
	api.RegisterPoolServer(rpcServer, &engine.Pool{})
	reflection.Register(rpcServer)
	if err := rpcServer.Serve(listener); err != nil {
		log.Fatalln("服务启动失败")
	}
}
