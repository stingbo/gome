package engine

import (
	"context"
	"fmt"
	"gome/gomengine/redis"
	"gome/gomengine/util"
	"strconv"
)

const (
	ADD int8 = 1
	DEL int8 = 2
)

var ctx = context.Background()
var cache = redis.NewRedisClient()

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
	fmt.Printf("%#v\n", depths)
	fmt.Printf("depths长度%#v\n", len(depths))
	//fmt.Printf("%T\n", depths)

	// 撮合计算逻辑
	if len(depths) > 0 {
		node := Match(&node, depths)
		fmt.Printf("匹配完之后的node--------%#v\n", node)
		if node.Volume <= 0 {
			return true
		}
	}
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
		fmt.Printf("匹配的价格信息--------%#v\n", v)
		price, _ := strconv.ParseFloat(v[0], 64)
		nodelink := OrderNode{} //copy一个新的节点
		nodelink = *node
		nodelink.Price = price
		nodelink.SetDepthHashKey()
		nodelink.SetNodeLink()
		fmt.Printf("去使用的nodelink信息--------%#v\n", nodelink)
		link := NodeLink{Node: &nodelink, Current: &nodelink} // 使用新的节点链去匹配计算
		node = MatchOrder(node, &link)
		if node.Volume <= 0 {
			break
		}
	}

	return node
}

func MatchOrder(node *OrderNode, link *NodeLink) *OrderNode {
	matchNode := link.GetFirstNode()
	fmt.Printf("第一个匹配的node--------%#v\n", matchNode)
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

		//fmt.Printf("撮合1node------：%#v\n", node)
		//fmt.Printf("撮合1match node------：%#v\n", matchNode)
		util.Info.Printf("撮合1node------：%#v\n", node)
		util.Info.Printf("撮合1match node------：%#v\n", matchNode)
		// 撮合成功通知
		//event(MatchEvent(node, matchNode, matchVolume))

		// 递归匹配
		MatchOrder(node, link)
	case diff == 0:
		matchVolume := matchNode.Volume
		node.Volume = node.Volume - matchVolume
		link.DeleteLinkNode(matchNode)
		DeletePoolMatchOrder(matchNode)

		//fmt.Printf("撮合2node------：%#v\n", node)
		//fmt.Printf("撮合2match node------：%#v\n", matchNode)
		util.Info.Printf("撮合2node------：%#v\n", node)
		util.Info.Printf("撮合2match node------：%#v\n", matchNode)
		// 撮合成功通知
		//event(MatchEvent(node, matchNode, matchVolume))
	case diff < 0:
		matchVolume := node.Volume
		matchNode.Volume = matchNode.Volume - matchVolume
		link.SetLinkNode(matchNode, matchNode.NodeName)

		updateNode := *matchNode // 更新委托池信息使用，不能直接使用matchNode，因为volume是剩余的，不是要减去的
		updateNode.Volume = matchVolume
		DeletePoolMatchOrder(&updateNode)
		node.Volume = 0

		//fmt.Printf("撮合3node------：%#v\n", node)
		//fmt.Printf("撮合3match node------：%#v\n", matchNode)
		//fmt.Printf("撮合3update node------：%#v\n", updateNode)
		util.Info.Printf("撮合3node------：%#v\n", node)
		util.Info.Printf("撮合3match node------：%#v\n", matchNode)
		util.Info.Printf("撮合3update node------：%#v\n", updateNode)
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
