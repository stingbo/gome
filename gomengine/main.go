package main

import (
	"context"
	"fmt"
	"gome/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"strings"
	"gome/gomengine/RabbitMQ"
)

type Order struct{}

func (fd *Order) DoOrder(ctx context.Context, in *api.OrderRequest) (out *api.OrderResponse, err error) {
	str := in.Symbol
	out = &api.OrderResponse{Message: strings.ToUpper(str)+" sting_bo"}


	rabbitmq := RabbitMQ.NewSimpleRabbitMQ("doOrder")

	rabbitmq.PublishSimple(str)

	return out, nil
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	listener := getRpcListener()

	rpcServer := grpc.NewServer()
	api.RegisterOrderServer(rpcServer, &Order{})
	reflection.Register(rpcServer)
	if err := rpcServer.Serve(listener); err != nil {
		log.Fatalln("服务启动失败")
	}
}
