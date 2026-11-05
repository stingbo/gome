package engine

import (
	"context"
	"github.com/go-redis/redis/v8"
	"gome/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
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
	order := api.OrderRequest{Uuid: "", Oid: "", Symbol: request.Symbol, Transaction: api.TransactionType(request.Transaction), Price: 0, Volume: 0}
	node := NewOrderNode(&order)

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
			p, priceErr := strconv.ParseFloat(price, 64)
			v, volumeErr := strconv.ParseFloat(rawVolume, 64)
			if priceErr != nil || volumeErr != nil {
				return nil, status.Error(codes.Internal, "invalid order book depth")
			}
			if v > 0 {
				data = append(data, &api.Depth{P: p, V: v})
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

func (pl *Pool) SetPrePool(ctx context.Context) error {
	return cache.HSet(ctx, pl.Node.OrderHashKey, pl.Node.OrderHashField, 1).Err()
}

func (pl *Pool) DeletePrePool() {
	cache.HDel(ctx, pl.Node.OrderHashKey, pl.Node.OrderHashField)
}

// 放入价格点对应的深度池
func (pl *Pool) SetDepthLink() bool {
	link := &NodeLink{Node: pl.Node}
	first := link.GetFirstNode()
	if first.Oid == "" {
		link.InitOrderLink()

		return true
	}
	last := link.GetLast()
	if last.Oid == "" {
		panic("expects last node is not empty.")
	}
	link.SetLast()

	return true
}

// 从价格点对应的深度链删除
func (pl *Pool) DeleteDepthLink() bool {
	link := &NodeLink{Node: pl.Node, Current: pl.Node}
	current := link.GetCurrent()
	if current.Oid == "" {
		return false
	}
	link.DeleteLinkNode(current)

	return true
}

// 增加价格对应的委托量
func (pl *Pool) SetPoolDepthVolume() {
	cache.HIncrByFloat(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashField, pl.Node.Volume)
}

// 减少价格对应的委托量
func (pl *Pool) DeletePoolDepthVolume() {
	cache.HIncrByFloat(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashField, (pl.Node.Volume * -1))
}

// 设置价格列表
func (pl *Pool) SetPoolDepth() {
	cache.ZAdd(ctx, pl.Node.OrderListSortSetKey, &redis.Z{Score: pl.Node.Price, Member: pl.Node.Price})
}

// 从价格列表删除
func (pl *Pool) DeletePoolDepth() {
	res := cache.HGet(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashField)
	volumeStr := res.Val()
	volume, _ := strconv.ParseFloat(volumeStr, 64)
	if volume <= 0 {
		cache.ZRem(ctx, pl.Node.OrderListSortSetKey, pl.Node.Price)
	}
}

// 深度总条数
func (pl *Pool) GetDepthTotal() int64 {
	total := cache.ZCard(ctx, pl.Node.OrderListSortSetKey)

	return total.Val()
}

// 获取买卖的深度
func (pl *Pool) GetDoubleSideDepth(offset int64, count int64) {
}

// 获取反向深度列表.
func (pl *Pool) GetReverseDepth() []string {
	price := strconv.FormatFloat(pl.Node.Price, 'f', -1, 64)
	if pl.Node.Transaction == int32(api.TransactionType_SELL) {
		return cache.ZRevRangeByScore(ctx, pl.Node.OrderListSortSetRKey,
			&redis.ZRangeBy{Min: price, Max: "+inf"}).Val()
	}
	return cache.ZRangeByScore(ctx, pl.Node.OrderListSortSetRKey,
		&redis.ZRangeBy{Min: "-inf", Max: price}).Val()
}
