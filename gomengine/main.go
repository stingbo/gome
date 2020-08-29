package main

import (
	"context"
	"fmt"
	"gome/api"
	"gome/gomengine/engine"
	megrpc "gome/gomengine/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
)

const (
	_ = iota
	ADD
	DEL
)

type Order struct{}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	gomegrpc := megrpc.NewRpcListener()
	listener := gomegrpc.Listener

	rpcServer := grpc.NewServer()
	api.RegisterOrderServer(rpcServer, &Order{})
	reflection.Register(rpcServer)
	if err := rpcServer.Serve(listener); err != nil {
		log.Fatalln("服务启动失败")
	}
}

func (fd *Order) DoOrder(ctx context.Context, request *api.OrderRequest) (response *api.OrderResponse, err error) {
	// 实例化撮合所需要的node
	orderNode := engine.NewOrderNode(*request)
	orderNode.Action = ADD
	// 放入预热池
	pool := engine.Pool{Node: orderNode}
	pool.SetPrePool()

	// 下单队列
	engine.PublishNewOrder(*orderNode)
	response = &api.OrderResponse{Message: "下单执行成功"}

	return response, nil
}

func (fd *Order) DeleteOrder(ctx context.Context, request *api.OrderRequest) (response *api.OrderResponse, err error) {
	// 实例化撮合所需要的node
	orderNode := engine.NewOrderNode(*request)
	orderNode.Action = DEL

	// 删除队列
	engine.PublishNewOrder(*orderNode)
	response = &api.OrderResponse{Message: "删除执行开始成功"}

	return response, nil
}
