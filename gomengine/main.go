package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gome/api"
	"gome/gomengine/Engine"
	"gome/gomengine/RabbitMQ"
	"gome/gomengine/gRPC"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
)

type Order struct{}

func (fd *Order) DoOrder(ctx context.Context, request *api.OrderRequest) (response *api.OrderResponse, err error) {
	// 实例化撮合所需要的node
	orderNode := Engine.NewOrderNode(*request)
	// 放入预热池
	pool := Engine.Pool{Node: *orderNode}
	pool.SetPrePool()

	// 下单队列
	order, err := json.Marshal(orderNode)
	rabbitmq := RabbitMQ.NewSimpleRabbitMQ("doOrder")
	rabbitmq.PublishSimple(order)
	response = &api.OrderResponse{Message: "下单成功"}

	return response, nil
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	gomegrpc := gRPC.NewRpcListener()
	listener := gomegrpc.Listener

	rpcServer := grpc.NewServer()
	api.RegisterOrderServer(rpcServer, &Order{})
	reflection.Register(rpcServer)
	if err := rpcServer.Serve(listener); err != nil {
		log.Fatalln("服务启动失败")
	}
}
