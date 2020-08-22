package Engine

import (
	"context"
	"fmt"
	"gome/gomengine/Redis"
)

const (
	ADD int8 = 1
	DEL int8 = 2
)

var ctx = context.Background()
var redis = Redis.NewRedisClient()

func DoOrder(node OrderNode) bool {
	if node.Action == ADD {
		Match(node)
	} else if node.Action == DEL {
		DeleteOrder(node)
	}

	return true
}

func Match(node OrderNode) bool {
	pool := &Pool{Node: node}
	if false == pool.ExistsPrePool() {
		return false
	}

	pool.DeletePrePool()

	// 撮合计算逻辑
	fmt.Printf("%#v\n", node)
	fmt.Printf("%T\n", node)

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

	link := &NodeLink{Node: node, Current: &node}
	nodelink := link.GetLinkNode(node.NodeName)
	fmt.Printf("删除时------：%#v\n", nodelink)
	fmt.Printf("删除时------：%T\n", nodelink)
	if nodelink.Oid == "" {
		return false
	}
	// 防止部分成交，删除过多委托量
	pool.Node.Volume = nodelink.Volume

	// 二，深度列表、数量更新、节点更新
	pool.DeletePoolDepthVolume()
	pool.DeletePoolDepth()

	// 三，从节点链里删除
	link.DeleteLinkNode(nodelink)

	return true
}
