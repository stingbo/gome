package engine

import "github.com/stingbo/gome/api"

type OrderNode struct {
	CommandID   string `json:",omitempty"`
	Action      int8   // 节点操作类型，1-add，2-del
	Uuid        string // 用户唯一标识
	Oid         string // 订单唯一标识
	Symbol      string // 交易对
	Transaction int32  // 交易方向，buy/sale
	Price       ticks  // 交易价格
	Volume      ticks  // 交易数量
	Accuracy    int    // 计算精度
	NodeName    string // 节点
	IsFirst     bool   // 是否是起始节点
	IsLast      bool   // 是否是结束节点
	PrevNode    string // 前一个节点
	NextNode    string // 后一个节点
	NodeLink    string // 节点链标识

	// hash对比池标识.
	OrderHashKey   string
	OrderHashField string

	// 有序集合委托列表.
	OrderListSortSetKey  string
	OrderListSortSetRKey string // 相反的委托

	// hash委托深度.
	OrderDepthHashKey   string
	OrderDepthHashField string
}

func newOrderNode(order *api.OrderRequest, action int8, accuracy int, price, volume ticks) *OrderNode {
	node := &OrderNode{
		Action: action, Uuid: order.Uuid, Oid: order.Oid, Symbol: order.Symbol,
		Transaction: int32(order.Transaction), Accuracy: accuracy, Price: price, Volume: volume,
	}
	node.SetOrderHashKey()
	node.SetListSortSetKey()
	node.SetDepthHashKey()
	node.SetNodeName()
	node.SetNodeLink()
	return node
}

func (node *OrderNode) SetOrderHashKey() {
	node.OrderHashKey = node.Symbol + ":comparison"
	node.OrderHashField = node.Symbol + ":" + node.Uuid + ":" + node.Oid
}

func (node *OrderNode) SetListSortSetKey() {
	if api.TransactionType_value["SELL"] == node.Transaction {
		node.OrderListSortSetKey = node.Symbol + ":SELL"
		node.OrderListSortSetRKey = node.Symbol + ":BUY"
	} else {
		node.OrderListSortSetKey = node.Symbol + ":BUY"
		node.OrderListSortSetRKey = node.Symbol + ":SELL"
	}
}

func (node *OrderNode) SetDepthHashKey() {
	node.OrderDepthHashKey = node.Symbol + ":depth"
	priceStr := node.Price.String()
	node.OrderDepthHashField = node.Symbol + ":depth:" + priceStr
}

func (node *OrderNode) SetNodeName() {
	node.NodeName = node.Symbol + ":node:" + node.Oid
}

func (node *OrderNode) SetNodeLink() {
	priceStr := node.Price.String()
	node.NodeLink = node.Symbol + ":link:" + priceStr
}
