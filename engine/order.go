package engine

import (
	"context"
	"gome/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"math"
	"regexp"
	"strings"
)

type Order struct{}

func (o Order) DoOrder(ctx context.Context, request *api.OrderRequest) (*api.OrderResponse, error) {
	if err := validateOrder(request, true); err != nil {
		return nil, err
	}
	// 实例化撮合所需要的node
	orderNode := NewOrderNode(request)
	orderNode.Action = ADD
	// 放入预热池
	pool := Pool{Node: orderNode}
	if err := pool.SetPrePool(ctx); err != nil {
		return nil, status.Error(codes.Unavailable, "order book unavailable")
	}

	// 下单队列
	if err := publish(ctx, orderNode.Symbol, orderNode); err != nil {
		log.Printf("publish order command for %s: %v", orderNode.Symbol, err)
		return nil, status.Error(codes.Unavailable, "order queue unavailable")
	}
	response := &api.OrderResponse{Message: "下单请求已入队"}

	return response, nil
}

func (o Order) DeleteOrder(ctx context.Context, request *api.OrderRequest) (*api.OrderResponse, error) {
	if err := validateOrder(request, false); err != nil {
		return nil, err
	}
	// 实例化撮合所需要的node
	orderNode := NewOrderNode(request)
	orderNode.Action = DEL

	// 删除队列
	if err := publish(ctx, orderNode.Symbol, orderNode); err != nil {
		log.Printf("publish order command for %s: %v", orderNode.Symbol, err)
		return nil, status.Error(codes.Unavailable, "order queue unavailable")
	}
	response := &api.OrderResponse{Message: "撤单请求已入队"}

	return response, nil
}

var symbolPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

func validateOrder(request *api.OrderRequest, add bool) error {
	if request == nil || !symbolPattern.MatchString(request.Symbol) {
		return status.Error(codes.InvalidArgument, "invalid symbol")
	}
	for _, id := range []string{request.Uuid, request.Oid} {
		if strings.TrimSpace(id) == "" || len(id) > 128 || strings.Contains(id, ":") {
			return status.Error(codes.InvalidArgument, "invalid user or order ID")
		}
	}
	if request.Transaction != api.TransactionType_BUY && request.Transaction != api.TransactionType_SELL {
		return status.Error(codes.InvalidArgument, "invalid order side")
	}
	scale := math.Pow10(Conf.MeConf.Accuracy)
	if !(request.Price > 0 && request.Price*scale <= (1<<53)-1) {
		return status.Error(codes.InvalidArgument, "price must be positive and within range")
	}
	if add && !(request.Volume > 0 && request.Volume*scale <= (1<<53)-1) {
		return status.Error(codes.InvalidArgument, "volume must be positive and within range")
	}
	return nil
}
