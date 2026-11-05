package engine

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/stingbo/gome/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

type Pool struct {
	Node *OrderNode
}

// 获取深度列表.
func (pl *Pool) GetDepth(requestCtx context.Context, request *api.DepthRequest) (*api.DepthResponse, error) {
	if request == nil || !symbolPattern.MatchString(request.Symbol) {
		return nil, status.Error(codes.InvalidArgument, "invalid symbol")
	}
	if request.Transaction != api.DepthRequest_BUY && request.Transaction != api.DepthRequest_SELL {
		return nil, status.Error(codes.InvalidArgument, "only single-side depth is supported")
	}
	node := OrderNode{Symbol: request.Symbol, Transaction: int32(request.Transaction)}
	node.SetListSortSetKey()
	node.SetDepthHashKey()

	total, err := cache.ZCard(requestCtx, node.OrderListSortSetKey).Result()
	if err != nil {
		return nil, status.Error(codes.Unavailable, "order book unavailable")
	}
	offset := request.Offset
	// 偏移量只能从0开始
	if offset < 0 {
		offset = 0
	}
	// 每次获取1~100条数据
	count := request.Count
	if count <= 0 || count > 100 {
		count = 20
	}
	rangeBy := redis.ZRangeBy{Min: "-inf", Max: "+inf", Offset: offset, Count: count}
	var res *redis.StringSliceCmd
	if request.Transaction == api.DepthRequest_SELL {
		res = cache.ZRangeByScore(requestCtx, node.OrderListSortSetKey, &rangeBy)
	} else {
		res = cache.ZRevRangeByScore(requestCtx, node.OrderListSortSetKey, &rangeBy)
	}
	prices, err := res.Result()
	if err != nil {
		return nil, status.Error(codes.Unavailable, "order book unavailable")
	}

	var data []*api.Depth
	if len(prices) > 0 {
		fields := make([]string, len(prices))
		for i, price := range prices {
			fields[i] = node.OrderDepthHashKey + ":" + price
		}
		volumes, err := cache.HMGet(requestCtx, node.OrderDepthHashKey, fields...).Result()
		if err != nil {
			return nil, status.Error(codes.Unavailable, "order book unavailable")
		}
		data = make([]*api.Depth, 0, len(prices))
		for i, price := range prices {
			rawVolume, ok := volumes[i].(string)
			if !ok {
				continue
			}
			p, priceErr := parseTicks(price)
			v, volumeErr := parseDepthTicks(rawVolume)
			if priceErr != nil || volumeErr != nil {
				return nil, status.Error(codes.Internal, "invalid order book depth")
			}
			if v.Sign() > 0 {
				volume, _ := new(big.Float).SetInt(v).Float64()
				data = append(data, &api.Depth{P: float64(p), V: volume})
			}
		}
	}

	response := &api.DepthResponse{
		Code:    0,
		Message: "获取成功",
		Total:   total,
		Data:    data,
	}

	return response, nil
}
