package Engine

import (
	"fmt"
	redis2 "github.com/go-redis/redis/v8"
	"gome/api"
	"strconv"
)

type Pool struct {
	Node *OrderNode
}

func (pl *Pool) SetPrePool() {
	redis.HSet(ctx, pl.Node.OrderHashKey, pl.Node.OrderHashField, 1)
}

func (pl *Pool) ExistsPrePool() bool {
	exists := redis.HExists(ctx, pl.Node.OrderHashKey, pl.Node.OrderHashField)

	return exists.Val()
}

func (pl *Pool) DeletePrePool() {
	if true == pl.ExistsPrePool() {
		redis.HDel(ctx, pl.Node.OrderHashKey, pl.Node.OrderHashField)
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
	redis.HIncrByFloat(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashField, pl.Node.Volume)
}

// 减少价格对应的委托量
func (pl *Pool) DeletePoolDepthVolume() {
	redis.HIncrByFloat(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashField, (pl.Node.Volume * -1))
}

// 设置价格列表
func (pl *Pool) SetPoolDepth() {
	redis.ZAdd(ctx, pl.Node.OrderListZsetKey, &redis2.Z{Score: pl.Node.Price, Member: pl.Node.Price})
}

// 从价格列表删除
func (pl *Pool) DeletePoolDepth() {
	res := redis.HGet(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashField)
	volumeStr := res.Val()
	volume, _ := strconv.ParseFloat(volumeStr, 64)
	if volume <= 0 {
		redis.ZRem(ctx, pl.Node.OrderListZsetKey, pl.Node.Price)
	}
}

// 获取反向深度列表.
func (pl *Pool) GetReverseDepth() [][]string {
	var depths [][]string
	price := strconv.FormatFloat(pl.Node.Price, 'f', -1, 64)
	if api.TransactionType_value["SALE"] == pl.Node.Transaction {
		rangeby := redis2.ZRangeBy{Min: price, Max: "+inf"}
		res := redis.ZRevRangeByScore(ctx, pl.Node.OrderListZsetRKey, &rangeby)
		fmt.Println("------------", res)
		prices := res.Val()
		for k, v := range prices {
			volres := redis.HGet(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashKey+":"+v)
			data := []string{v, volres.Val()}
			depths = append(depths, data)
			fmt.Println("buy取出的结果------------", k, v)
		}
	} else {
		rangeby := redis2.ZRangeBy{Min: "-inf", Max: price}
		res := redis.ZRangeByScore(ctx, pl.Node.OrderListZsetRKey, &rangeby)
		fmt.Println("------------", res)
		prices := res.Val()
		for k, v := range prices {
			volres := redis.HGet(ctx, pl.Node.OrderDepthHashKey, pl.Node.OrderDepthHashKey+":"+v)
			data := []string{v, volres.Val()}
			depths = append(depths, data)

			fmt.Println("sale取出的结果------------", k, v)
		}
	}

	return depths
}
