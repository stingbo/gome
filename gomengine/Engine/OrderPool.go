package Engine

import (
	"context"
	"gome/gomengine/Redis"
)

var ctx = context.Background()
var redis = Redis.NewRedisClient()

func SetPrePool(order OrderNode) {
	redis.HSet(ctx, order.OrderHashKey, order.OrderHashField, 1)
}

func ExistsPrePool(order OrderNode) bool {
	exists := redis.HExists(ctx, order.OrderHashKey, order.OrderHashField)
	//fmt.Printf("%#v\n",exists)   // main.point{x:1, y:2}
	//fmt.Printf("%T\n",exists)
	//cacheOrder, _ := json.Marshal(order)
	//redis.HSet(ctx, order.NodeLink, order.Node, cacheOrder)

	return exists.Val()
}

func DeletePrePool(order OrderNode) {
	if true == ExistsPrePool(order) {
		redis.HDel(ctx, order.OrderHashKey, order.OrderHashField)
	}
}
