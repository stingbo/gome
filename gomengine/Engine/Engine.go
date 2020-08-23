package Engine

import (
	"context"
	"fmt"
	"gome/gomengine/Redis"
	"strconv"
)

const (
	ADD int8 = 1
	DEL int8 = 2
)

var ctx = context.Background()
var redis = Redis.NewRedisClient()

func DoOrder(node OrderNode) bool {
	if node.Action == ADD {
		SetOrder(node)
	} else if node.Action == DEL {
		DeleteOrder(node)
	}

	return true
}

func SetOrder(node OrderNode) bool {
	pool := &Pool{Node: &node}
	if false == pool.ExistsPrePool() {
		return false
	}

	pool.DeletePrePool()
	depths := pool.GetReverseDepth()
	//fmt.Printf("%#v\n", depths)
	//fmt.Printf("%T\n", depths)

	if len(depths) > 0 {
		//fmt.Printf("depths长度%#v\n", len(depths))
		node := Match(&node, depths)
		if node.Volume <= 0 {
			return true
		}
	}

	// 撮合计算逻辑
	//fmt.Printf("%#v\n", node)
	//fmt.Printf("%T\n", node)

	// 深度列表、数量更新、节点更新
	pool.SetPoolDepth()
	pool.SetPoolDepthVolume()
	pool.SetDepthLink()

	return true
}

func DeleteOrder(node OrderNode) bool {
	// 一，从标识池删除，避免队列有积压时未消费问题
	pool := &Pool{Node: &node}
	pool.DeletePrePool()

	link := &NodeLink{Node: &node, Current: &node}
	nodelink := link.GetLinkNode(node.NodeName)
	//fmt.Printf("删除时------：%#v\n", nodelink)
	//fmt.Printf("删除时------：%T\n", nodelink)
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

func Match(node *OrderNode, depths [][]string) *OrderNode {
	for _, v := range depths {
		price, _ := strconv.ParseFloat(v[0], 64)
		nodelink := node
		nodelink.Price = price
		link := NodeLink{Node: nodelink, Current: nodelink}
		node = MatchOrder(node, &link)
	}

	return node
}

func MatchOrder(node *OrderNode, link *NodeLink) *OrderNode {
	matchNode := link.GetFirstNode()
	if matchNode.Oid == "" {
		return node
	}
	diff := node.Volume - matchNode.Volume
	switch {
	case diff > 0:
		matchVolume := matchNode.Volume
		node.Volume = node.Volume - matchVolume
		link.DeleteLinkNode(matchNode)
		DeletePoolMatchOrder(matchNode)

		fmt.Printf("撮合node------：%#v\n", node)
		fmt.Printf("撮合match node------：%#v\n", matchNode)
		// 撮合成功通知
		//event(MatchEvent(node, matchNode, matchVolume))

		// 递归匹配
		MatchOrder(node, link)
	case diff == 0:
		matchVolume := matchNode.Volume
		node.Volume = node.Volume - matchVolume
		link.DeleteLinkNode(matchNode)
		DeletePoolMatchOrder(matchNode)

		fmt.Printf("撮合node------：%#v\n", node)
		fmt.Printf("撮合match node------：%#v\n", matchNode)
		// 撮合成功通知
		//event(MatchEvent(node, matchNode, matchVolume))
	case diff < 0:
		matchVolume := node.Volume
		matchNode.Volume = matchNode.Volume - matchVolume
		link.SetLinkNode(matchNode, matchNode.NodeLink)
		DeletePoolMatchOrder(node)
		node.Volume = 0

		fmt.Printf("撮合node------：%#v\n", node)
		fmt.Printf("撮合match node------：%#v\n", matchNode)
		// 撮合成功通知
		//event(MatchEvent(node, matchNode, matchVolume))
	}

	return node
}

func DeletePoolMatchOrder(node *OrderNode) {
	pool := &Pool{Node: node}

	// 二，深度列表、数量更新、节点更新
	pool.DeletePoolDepthVolume()
	pool.DeletePoolDepth()
}
