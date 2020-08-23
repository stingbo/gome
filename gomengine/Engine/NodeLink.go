package Engine

import (
	"encoding/json"
)

type NodeLink struct {
	Current *OrderNode
	Node    *OrderNode
}

func (nl *NodeLink) InitOrderLink() {
	nl.Node.IsFirst = true
	nl.Node.IsLast = true

	nl.SetFristPointer(nl.Node.NodeName)
	nl.SetLastPointer(nl.Node.NodeName)
	nl.SetLinkNode(nl.Node, nl.Node.NodeName)
}

func (nl *NodeLink) GetLinkNode(nodeName string) *OrderNode {
	res := redis.HGet(ctx, nl.Node.NodeLink, nodeName)
	if res.Val() == "" {
		return &OrderNode{}
	}

	node := &OrderNode{}
	json.Unmarshal([]byte(res.Val()), &node)
	nl.Current = node // 获取某个节点时，把此节点设置为当前节点

	return node
}

func (nl *NodeLink) SetFristPointer(nodename string) {
	redis.HSet(ctx, nl.Node.NodeLink, "f", nodename)
}

func (nl *NodeLink) GetFirstNode() *OrderNode {
	res := redis.HGet(ctx, nl.Node.NodeLink, "f")
	if res.Val() == "" {
		return &OrderNode{}
	}

	order := nl.GetLinkNode(res.Val())
	if order.Uuid != "" {
		return order
	}
	nl.Current = order

	return order
}

func (nl *NodeLink) SetLast() {
	nl.GetLast()
	nl.Current.IsLast = false
	nl.Current.NextNode = nl.Node.NodeName
	nl.SetLinkNode(nl.Current, nl.Current.NodeName) // 更新本身节点信息

	nl.Node.PrevNode = nl.Current.NodeName // 重置last节点信息
	nl.SetLastPointer(nl.Node.NodeName)

	nl.Node.IsLast = true
	nl.SetLinkNode(nl.Node, nl.Node.NodeName)
}

func (nl *NodeLink) SetLastPointer(nodename string) {
	redis.HSet(ctx, nl.Node.NodeLink, "l", nodename)
}

func (nl *NodeLink) GetLast() *OrderNode {
	res := redis.HGet(ctx, nl.Node.NodeLink, "l")
	if res.Val() == "" {
		return &OrderNode{}
	}

	order := nl.GetLinkNode(res.Val())
	if order.Uuid == "" {
		return order
	}
	nl.Current = order

	return order
}

func (nl *NodeLink) GetCurrent() *OrderNode {
	return nl.Current
}

func (nl *NodeLink) GetPrev() *OrderNode {
	current := nl.GetCurrent()
	prevName := current.PrevNode
	if prevName == "" {
		return &OrderNode{}
	}
	node := nl.GetLinkNode(prevName)
	if node.Oid == "" {
		return &OrderNode{}
	}
	//nl.Current = node //是否需要重置当前节点?

	return node
}

func (nl *NodeLink) GetNext() *OrderNode {
	current := nl.GetCurrent()
	nextName := current.NextNode
	if nextName == "" {
		return &OrderNode{}
	}
	node := nl.GetLinkNode(nextName)
	if node.Oid == "" {
		return &OrderNode{}
	}
	//nl.Current = node //是否需要重置当前节点?

	return node
}

func (nl *NodeLink) SetLinkNode(order *OrderNode, nodeName string) {
	orderJson, _ := json.Marshal(order)
	redis.HSet(ctx, nl.Node.NodeLink, nodeName, orderJson)
}

func (nl *NodeLink) DeleteLinkNode(node *OrderNode) {
	if node.IsFirst && node.IsLast {
		redis.HDel(ctx, node.NodeLink, "f")
		redis.HDel(ctx, node.NodeLink, "l")
		redis.HDel(ctx, node.NodeLink, node.NodeName)
	} else if node.IsFirst && !node.IsLast {
		next := nl.GetNext()
		if next.Oid == "" {
			panic("expects next node is not empty.")
		}
		redis.HDel(ctx, node.NodeLink, node.NodeName)
		next.IsFirst = true
		next.PrevNode = ""
		nl.SetFristPointer(next.NodeName)
		nl.SetLinkNode(next, next.NodeName)
	} else if !node.IsFirst && node.IsLast {
		prev := nl.GetPrev()
		if prev.Oid == "" {
			panic("expects prev node is not empty.")
		}
		redis.HDel(ctx, node.NodeLink, node.NodeName)
		prev.IsLast = true
		prev.NextNode = ""
		nl.SetLastPointer(prev.NodeName)
		nl.SetLinkNode(prev, prev.NodeName)
	} else {
		prev := nl.GetPrev()
		current := nl.GetNext()
		next := nl.GetNext()
		//fmt.Printf("删除时prev------：%#v\n", prev)
		//fmt.Printf("删除时next------：%#v\n", next)

		if prev.Oid == "" && next.Oid == "" {
			panic("expects relation node is not empty.")
		}
		redis.HDel(ctx, current.NodeLink, current.NodeName)

		prev.NextNode = next.NodeName
		next.PrevNode = prev.NodeName
		nl.SetLinkNode(prev, prev.NodeName)
		nl.SetLinkNode(next, next.NodeName)
	}
}
