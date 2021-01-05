package engine

import (
	"context"
	"gome/api"
)

type Order struct{}

func (o Order) DoOrder(ctx2 context.Context, request *api.OrderRequest) (*api.OrderResponse, error) {
	// 实例化撮合所需要的node
	orderNode := NewOrderNode(*request)
	orderNode.Action = ADD
	// 放入预热池
	pool := Pool{Node: orderNode}
	pool.SetPrePool()

	// 下单队列
	PublishNewOrder(*orderNode)
	response := &api.OrderResponse{Message: "下单执行成功"}

	return response, nil
}

func (o Order) DeleteOrder(ctx2 context.Context, request *api.OrderRequest) (*api.OrderResponse, error) {
	// 实例化撮合所需要的node
	orderNode := NewOrderNode(*request)
	orderNode.Action = DEL

	// 删除队列
	PublishNewOrder(*orderNode)
	response := &api.OrderResponse{Message: "删除执行开始成功"}

	return response, nil
}

