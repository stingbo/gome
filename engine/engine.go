package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"gome/redis"
	"gome/util"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"strconv"
	"time"
)

const (
	_ = iota
	ADD
	DEL
	timeFormat = "2006-01-02 15:04:05"
)

var (
	ctx = context.Background()
	cache = redis.NewRedisClient()
	Conf *util.MeConfig
	Debug bool
	LogLevel string
)

type MatchResult struct {
	Action      int8    // 通知类型，1-match;2-del
	Node        OrderNode
	MatchNode   OrderNode
	MatchVolume float64
	MatchTime   string
}

func init() {
	confFile, _ := ioutil.ReadFile("config.yaml")
	yaml.Unmarshal(confFile, &Conf)
	Debug = Conf.Debug
	LogLevel = Conf.LogLevel
}

func PublishNewOrder(node OrderNode) bool {
	if Debug {
		fmt.Printf("来源数据----------:%#v\n", node)
	}
	symbol := node.Symbol
	order, err := json.Marshal(node)
	mq := NewSimpleRabbitMQ(symbol)
	mq.PublishNewOrder(order)
	if err != nil {
		return false
	}

	return true
}

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
	if Debug {
		fmt.Printf("depths长度----------:%#v\n", len(depths))
		fmt.Printf("depths数据----------:%#v\n", depths)
	}

	// 撮合计算逻辑
	if len(depths) > 0 {
		node := Match(&node, depths)
		if node.Volume <= 0 {
			return true
		}
	}

	// 深度列表、数量更新、节点更新
	pool.SetPoolDepth()
	pool.SetPoolDepthVolume()
	pool.SetDepthLink()

	return true
}

func DeleteOrder(node OrderNode) bool {
	symbol := node.Symbol
	noticeSymbol := "notice:"+symbol

	// 一，从标识池删除，避免队列有积压时未消费问题
	pool := &Pool{Node: &node}
	pool.DeletePrePool()

	link := &NodeLink{Node: &node, Current: &node}
	nodeLink := link.GetLinkNode(node.NodeName)
	if Debug {
		fmt.Printf("删除节点信息----------:%#v\n", nodeLink)
	}
	if nodeLink.Oid == "" {
		return false
	}
	if nodeLink.Uuid != node.Uuid {
		fmt.Printf("删除节点用户标识信息不匹配----------:%#v\n", node)
		return false
	}
	if nodeLink.Transaction != node.Transaction {
		fmt.Printf("删除节点交易方向信息不匹配----------:%#v\n", node)
		return false
	}

	// 防止部分成交，删除过多委托量
	pool.Node.Volume = nodeLink.Volume

	// 二，深度列表、数量更新、节点更新
	pool.DeletePoolDepthVolume()
	pool.DeletePoolDepth()

	// 三，从节点链里删除
	link.DeleteLinkNode(nodeLink)

	// 撤单通知
	matchResult := MatchResult{
		Action: DEL,
		Node: node,
		MatchNode: node,
		MatchVolume: 0,
		MatchTime: time.Now().Format(timeFormat),
	}
	match, _ := json.Marshal(matchResult)
	mq := NewSimpleRabbitMQ(noticeSymbol)
	mq.PublishNewOrder(match)

	return true
}

func Match(node *OrderNode, depths [][]string) *OrderNode {
	for _, v := range depths {
		price, _ := strconv.ParseFloat(v[0], 64)
		nodeLink := OrderNode{} //copy一个新的节点
		nodeLink = *node
		nodeLink.Price = price
		nodeLink.SetDepthHashKey()
		nodeLink.SetNodeLink()
		if Debug {
			fmt.Printf("匹配的价格信息----------:%#v\n", v)
			fmt.Printf("去使用的nodelink信息--------%#v\n", nodeLink)
		}
		link := NodeLink{Node: &nodeLink, Current: &nodeLink} // 使用新的节点链去匹配计算
		node = MatchOrder(node, &link)
		if node.Volume <= 0 {
			break
		}
	}

	return node
}

func MatchOrder(node *OrderNode, link *NodeLink) *OrderNode {
	symbol := node.Symbol
	noticeSymbol := "notice:"+symbol
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

		// 撮合成功通知
		matchResult := MatchResult{
			Action: ADD,
			Node: *node,
			MatchNode: *matchNode,
			MatchVolume: matchVolume,
			MatchTime: time.Now().Format(timeFormat),
		}
		match, _ := json.Marshal(matchResult)
		mq := NewSimpleRabbitMQ(noticeSymbol)
		mq.PublishNewOrder(match)

		// 递归匹配
		MatchOrder(node, link)
	case diff == 0:
		matchVolume := matchNode.Volume
		node.Volume = node.Volume - matchVolume
		link.DeleteLinkNode(matchNode)
		DeletePoolMatchOrder(matchNode)

		// 撮合成功通知
		matchResult := MatchResult{
			Action: ADD,
			Node: *node,
			MatchNode: *matchNode,
			MatchVolume: matchVolume,
			MatchTime: time.Now().Format(timeFormat),
		}
		match, _ := json.Marshal(matchResult)
		mq := NewSimpleRabbitMQ(noticeSymbol)
		mq.PublishNewOrder(match)
	case diff < 0:
		matchVolume := node.Volume
		matchNode.Volume = matchNode.Volume - matchVolume
		link.SetLinkNode(matchNode, matchNode.NodeName)

		updateNode := *matchNode // 更新委托池信息使用，不能直接使用matchNode，因为volume是剩余的，不是要减去的
		updateNode.Volume = matchVolume
		DeletePoolMatchOrder(&updateNode)
		node.Volume = 0

		// 撮合成功通知
		matchResult := MatchResult{
			Action: ADD,
			Node: *node,
			MatchNode: *matchNode,
			MatchVolume: matchVolume,
			MatchTime: time.Now().Format(timeFormat),
		}
		match, _ := json.Marshal(matchResult)
		mq := NewSimpleRabbitMQ(noticeSymbol)
		mq.PublishNewOrder(match)
	}

	return node
}

func DeletePoolMatchOrder(node *OrderNode) {
	pool := &Pool{Node: node}

	// 二，深度列表、数量更新、节点更新
	pool.DeletePoolDepthVolume()
	pool.DeletePoolDepth()
}
