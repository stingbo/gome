package Engine

import (
	"gome/api"
	"math"
	"strconv"
)

type OrderNode struct {
	Uuid        string  // 用户唯一标识
	Oid         string  // 订单唯一标识
	Symbol      string  // 交易对
	Transaction int32   // 交易方向，buy/sale
	Price       float64 // 交易价格
	Volume      float64 // 交易数量
	Accuracy    float64 // 计算精度
	NodeName    string  // 节点
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
	node.SetAccuracy()
	node.SetUuid(order)
	node.SetOid(order)
	node.SetSymbol(order)
	node.SetTransaction(order)
	node.SetVolume(order)
	node.SetPrice(order)
	node.SetOrderHashKey()
	node.SetListZsetKey()
	node.SetDepthHashKey()
	node.SetNodeName()
	node.SetNodeLink()

	return node
}

func (node *OrderNode) SetAccuracy() {
	node.Accuracy = 8
}

func (node *OrderNode) SetUuid(order api.OrderRequest) {
	node.Uuid = order.Uuid
}

func (node *OrderNode) SetOid(order api.OrderRequest) {
	node.Oid = order.Oid
}

func (node *OrderNode) SetSymbol(order api.OrderRequest) {
	node.Symbol = order.Symbol
}

func (node *OrderNode) SetTransaction(order api.OrderRequest) {
	node.Transaction = int32(order.Transaction)
}

func (node *OrderNode) SetVolume(order api.OrderRequest) {
	node.Volume = order.Volume * math.Pow(10, node.Accuracy)
}

func (node *OrderNode) SetPrice(order api.OrderRequest) {
	node.Price = order.Price * math.Pow(10, node.Accuracy)
}

func (node *OrderNode) SetOrderHashKey() {
	node.OrderHashKey = node.Symbol + ":comparison"
	node.OrderHashField = node.Symbol + ":" + node.Uuid + ":" + node.Oid
}

func (node *OrderNode) SetListZsetKey() {
	node.OrderListZsetKey = node.Symbol + ":" + api.TransactionType_name[node.Transaction]
}

func (node *OrderNode) SetDepthHashKey() {
	node.OrderDepthHashKey = node.Symbol + ":depth"
	node.OrderDepthHashField = node.Symbol + ":depth:" + strconv.FormatFloat(node.Price, 'f', -1, 64)
}

func (node *OrderNode) SetNodeName() {
	node.NodeName = node.Symbol + ":node:" + node.Oid
}

func (node *OrderNode) SetNodeLink() {
	node.NodeLink = node.Symbol + ":link:" + strconv.FormatFloat(node.Price, 'f', -1, 64)
}
