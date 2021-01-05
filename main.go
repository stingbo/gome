package main

import (
	"context"
	"fmt"
	"gome/api"
	"gome/engine"
	rpc "gome/grpc"
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
type Pool struct{}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	rpcListener := rpc.NewRpcListener()
	listener := rpcListener.Listener

	rpcServer := grpc.NewServer()
	api.RegisterOrderServer(rpcServer, &Order{})
	api.RegisterPoolServer(rpcServer, &Pool{})
	reflection.Register(rpcServer)
	if err := rpcServer.Serve(listener); err != nil {
		log.Fatalln("服务启动失败")
	}
}

// 实现api order里定义的方法
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

// 实现api pool里定义的方法
func (fd *Pool) GetDepth(ctx context.Context, request *api.DepthRequest) (response *api.DepthResponse, err error) {
	order := api.OrderRequest{Uuid: "", Oid: "", Symbol: request.Symbol, Transaction: api.TransactionType(request.Transaction), Price: 0, Volume: 0}
	node := engine.NewOrderNode(order)
	pool := engine.Pool{Node: node}

	total := pool.GetDepthTotal()
	depths := pool.GetDepth(request.Offset, request.Count)

	var data []*api.Depth
	for _, d := range depths {
		depth := api.Depth{P: d["p"], V: d["v"]}
		data = append(data, &depth)
	}
	response = &api.DepthResponse{
		Code: 0,
		Message: "获取成功",
		Total: total,
		Data: data,
	}

	return response, nil
}
