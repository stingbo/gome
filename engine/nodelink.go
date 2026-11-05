package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// priceLevel stages the touched FIFO links in memory until the book commits.
type priceLevel struct {
	key     string
	first   string
	last    string
	changes map[string]*OrderNode
}

func loadLevel(ctx context.Context, tx *redis.Tx, key string) (*priceLevel, error) {
	if err := tx.Watch(ctx, key).Err(); err != nil {
		return nil, err
	}
	pointers, err := tx.HMGet(ctx, key, "f", "l").Result()
	if err != nil {
		return nil, err
	}
	first, _ := pointers[0].(string)
	last, _ := pointers[1].(string)
	if (first == "") != (last == "") {
		return nil, fmt.Errorf("inconsistent price level pointers at %s", key)
	}
	return &priceLevel{key: key, first: first, last: last, changes: make(map[string]*OrderNode)}, nil
}

func (level *priceLevel) get(ctx context.Context, tx *redis.Tx, name string) (*OrderNode, error) {
	if node, changed := level.changes[name]; changed {
		return node, nil
	}
	raw, err := tx.HGet(ctx, level.key, name).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var node OrderNode
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		return nil, err
	}
	if node.NodeName != name || node.NodeLink != level.key || node.Oid == "" || !(node.Volume > 0) {
		return nil, fmt.Errorf("invalid order node %s at %s", name, level.key)
	}
	return &node, nil
}

func (level *priceLevel) append(ctx context.Context, tx *redis.Tx, node OrderNode) error {
	node.IsFirst = level.first == ""
	node.IsLast = true
	node.PrevNode = level.last
	node.NextNode = ""
	if node.IsFirst {
		level.first = node.NodeName
	} else {
		tail, err := level.get(ctx, tx, level.last)
		if err != nil {
			return err
		}
		if tail == nil || tail.NextNode != "" || !tail.IsLast {
			return fmt.Errorf("invalid tail at %s", level.key)
		}
		tail.NextNode = node.NodeName
		tail.IsLast = false
		level.changes[tail.NodeName] = tail
	}
	level.last = node.NodeName
	level.changes[node.NodeName] = &node
	return nil
}

func (level *priceLevel) remove(ctx context.Context, tx *redis.Tx, node *OrderNode) error {
	if node.PrevNode == "" {
		if level.first != node.NodeName {
			return fmt.Errorf("invalid head at %s", level.key)
		}
		level.first = node.NextNode
	} else {
		prev, err := level.get(ctx, tx, node.PrevNode)
		if err != nil {
			return err
		}
		if prev == nil || prev == node || prev.NodeName == node.NodeName || prev.NextNode != node.NodeName {
			return fmt.Errorf("invalid predecessor of %s", node.NodeName)
		}
		prev.NextNode = node.NextNode
		prev.IsLast = node.NextNode == ""
		level.changes[prev.NodeName] = prev
	}
	if node.NextNode == "" {
		if level.last != node.NodeName {
			return fmt.Errorf("invalid tail at %s", level.key)
		}
		level.last = node.PrevNode
	} else {
		next, err := level.get(ctx, tx, node.NextNode)
		if err != nil {
			return err
		}
		if next == nil || next.NodeName == node.NodeName || next.PrevNode != node.NodeName {
			return fmt.Errorf("invalid successor of %s", node.NodeName)
		}
		next.PrevNode = node.PrevNode
		next.IsFirst = node.PrevNode == ""
		level.changes[next.NodeName] = next
	}
	level.changes[node.NodeName] = nil
	return nil
}

func (level *priceLevel) write(ctx context.Context, pipe redis.Pipeliner) error {
	if level.first == "" {
		pipe.Del(ctx, level.key)
		return nil
	}
	pipe.HSet(ctx, level.key, "f", level.first, "l", level.last)
	for name, node := range level.changes {
		if node == nil {
			pipe.HDel(ctx, level.key, name)
			continue
		}
		raw, err := json.Marshal(node)
		if err != nil {
			return err
		}
		pipe.HSet(ctx, level.key, name, raw)
	}
	return nil
}
