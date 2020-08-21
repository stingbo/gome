package Engine

import (
	"gome/api"
	"math"
)

type OrderNode struct {
	Uuid        string  // 用户唯一标识
	Oid         string  // 订单唯一标识
	Symbol      string  // 交易对
	Transaction int32  // 交易方向，buy/sale
	Price       float64 // 交易价格
	Volume      float64 // 交易数量
	Accuracy    float64   // 计算精度
	Node        string  // 节点
	IsFirst     bool    // 是否是起始节点
	IsLast      bool    // 是否是结束节点
	PrevNode    string  // 前一个节点
	NextNode    string  // 后一个节点
	NodeLink    string  // 节点链标识

	// hash对比池标识.
	OrderHashKey   string
	OrderHashField string

	// zset委托列表.
	OrderListZsetKey string

	// hash委托深度.
	OrderDepthHashKey   string
	OrderDepthHashField string
}

func NewOrderNode(order api.OrderRequest) *OrderNode {
	node := &OrderNode{}
	SetAccuracy(node)
	SetUuid(node, order)
	SetOid(node, order)
	SetSymbol(node, order)
	SetTransaction(node, order)
	SetVolume(node, order)
	SetPrice(node, order)
	SetOrderHashKey(node)
	SetListZsetKey(node)

	return node
}

func SetAccuracy(node *OrderNode) *OrderNode {
	node.Accuracy = 8

	return node
}

func SetUuid(node *OrderNode, order api.OrderRequest) *OrderNode {
	node.Uuid = order.Uuid

	return node
}

func SetOid(node *OrderNode, order api.OrderRequest) *OrderNode {
	node.Oid = order.Oid

	return node
}

func SetSymbol(node *OrderNode, order api.OrderRequest) *OrderNode {
	node.Symbol = order.Symbol

	return node
}

func SetTransaction(node *OrderNode, order api.OrderRequest) *OrderNode {
	node.Transaction = int32(order.Transaction)

	return node
}

func SetVolume(node *OrderNode, order api.OrderRequest) *OrderNode {
	node.Volume = order.Volume * math.Pow(10,node.Accuracy)

	return node
}

func SetPrice(node *OrderNode, order api.OrderRequest) *OrderNode {
	node.Price = order.Price * math.Pow(10,node.Accuracy)

	return node
}

func SetOrderHashKey(node *OrderNode) *OrderNode {
	node.OrderHashKey = node.Symbol+":comparison"
	node.OrderHashField = node.Symbol+":"+node.Uuid+":"+node.Oid;

	return node
}

func SetListZsetKey(node *OrderNode) *OrderNode {

	node.OrderListZsetKey = node.Symbol+":"+api.TransactionType_name[node.Transaction]

	return node
}