package Engine

import "gome/api"

type OrderNode struct {
	Uuid        string  // 用户唯一标识
	Oid         string  // 订单唯一标识
	Symbol      string  // 交易对
	Transaction string  // 交易方向，buy/sale
	Price       float64 // 交易价格
	Volume      float64 // 交易数量
	Accuracy    uint8   // 计算精度
	Node        string  // 节点
	IsFirst     bool    // 是否是起始节点
	IsLast      bool    // 是否是结束节点
	PrevNode    string  // 前一个节点
	NextNode    string  // 后一个节点
	NodeLink    string  // 节点链标识

	// hash对比池标识.
	OrderHashKey   string
	orderHashField string

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
