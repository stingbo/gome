package engine

import (
	"context"
	"github.com/go-redis/redis/v8"
	"gome/api"
	"strconv"
)

type Pool struct {
	Node *OrderNode
}

// 获取深度列表.
func (pl *Pool) GetDepth(ctx2 context.Context, request *api.DepthRequest) (*api.DepthResponse, error) {
	order := api.OrderRequest{Uuid: "", Oid: "", Symbol: request.Symbol, Transaction: api.TransactionType(request.Transaction), Price: 0, Volume: 0}
	node := NewOrderNode(order)
	pool := Pool{Node: node}

	total := pool.GetDepthTotal()
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
	res := cache.ZRevRangeByScore(ctx, pool.Node.OrderListSortSetKey, &rangeBy)
	prices := res.Val()

	var data []*api.Depth
	for _, p := range prices {
		vols := cache.HGet(ctx, pool.Node.OrderDepthHashKey, pool.Node.OrderDepthHashKey+":"+p)
		price, _ := strconv.ParseFloat(p, 64)
		volume, _ := strconv.ParseFloat(vols.Val(), 64)

		depth := api.Depth{P: price, V: volume}
		data = append(data, &depth)
	}

	response := &api.DepthResponse{
		Code: 0,
		Message: "获取成功",
		Total: total,
		Data: data,
	}

	return response, nil
}

func (pl *Pool) SetPrePool() {
	cache.HSet(ctx, pl.Node.OrderHashKey, pl.Node.OrderHashField, 1)
}

func (pl *Pool) ExistsPrePool() bool {
	exists := cache.HExists(ctx, pl.Node.OrderHashKey, pl.Node.OrderHashField)

	return exists.Val()
}

func (pl *Pool) DeletePrePool() {
	if true == pl.ExistsPrePool() {
		cache.HDel(ctx, pl.Node.OrderHashKey, pl.Node.OrderHashField)
	}
}

//放入价格点对应的深度池
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

//从价格点对应的深度链删除
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
func (pl *Pool) GetReverseDepth() [][]string {
	var depths [][]string
	price := strconv.FormatFloat(pl.Node.Price, 'f', -1, 64)
	if api.TransactionType_value["SELL"] == pl.Node.Transaction {
		rangeBy := redis.ZRangeBy{Min: price, Max: "+inf"}
		res := cache.ZRevRangeByScore(ctx, pl.Node.OrderListSortSetRKey, &rangeBy)
		prices := res.Val()
		for _, p := range prices {
			vols := cache.HGet(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashKey+":"+p)
			data := []string{p, vols.Val()}
			depths = append(depths, data)
		}
	} else {
		rangeBy := redis.ZRangeBy{Min: "-inf", Max: price}
		res := cache.ZRangeByScore(ctx, pl.Node.OrderListSortSetRKey, &rangeBy)
		prices := res.Val()
		for _, p := range prices {
			vols := cache.HGet(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashKey+":"+p)
			data := []string{p, vols.Val()}
			depths = append(depths, data)
		}
	}

	return depths
}
