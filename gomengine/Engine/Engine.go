package Engine

import (
	"context"
	"fmt"
	"gome/gomengine/Redis"
)

var ctx = context.Background()
var redis = Redis.NewRedisClient()

func Match(order OrderNode) bool {
	pool := &Pool{Node: order}
	if false == pool.ExistsPrePool() {
		return false
	}

	pool.DeletePrePool()

	// 撮合计算逻辑
	fmt.Printf("%#v\n", order)
	fmt.Printf("%T\n", order)

	// 深度列表、数量更新、节点更新
	pool.SetPoolDepth()
	pool.SetPoolDepthVolume()
	pool.SetDepthLink()

	return true
}

func DeleteOrder(node OrderNode) bool {
	// 一，从标识池删除，避免队列有积压时未消费问题
	pool := &Pool{Node: node}
	pool.DeletePrePool()

	link := &NodeLink{Node: node}
	nodelink := link.GetLinkNode(node.NodeName)
	if nodelink.Oid == "" {
		return false
	}
	pool.Node.Volume = nodelink.Volume

	// 二，深度列表、数量更新、节点更新
	pool.DeletePoolDepthVolume()
	pool.DeletePoolDepth()

	// 三，从节点链里删除
	link.DeleteLinkNode(&node)

	return true
}
