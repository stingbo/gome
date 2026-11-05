package engine

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gome/api"
	"gome/utils"
)

func TestOrderBook(t *testing.T) {
	port := os.Getenv("GOME_TEST_REDIS_PORT")
	if port == "" {
		t.Skip("set GOME_TEST_REDIS_PORT to an isolated Redis instance")
	}
	Configure(&utils.MeConfig{
		CacheConf: utils.Cache{Host: "127.0.0.1", Port: port},
		MeConf:    utils.Mengine{Accuracy: 8},
	})
	t.Cleanup(func() { cache.Close() })
	if err := cache.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	symbol := fmt.Sprintf("book-test-%d", time.Now().UnixNano())
	add := func(id string, price float64) *OrderNode {
		node := NewOrderNode(&api.OrderRequest{
			Uuid: "user", Oid: id, Symbol: symbol, Transaction: api.TransactionType_SELL, Price: price, Volume: 2,
		})
		if err := (&Pool{Node: node}).SetPrePool(ctx); err != nil {
			t.Fatal(err)
		}
		SetOrder(*node)
		return node
	}
	node := add("one", 1.25)
	add("one", 1.25)
	add("two", 1.5)
	first := (&NodeLink{Node: node}).GetFirstNode()
	if !first.IsFirst || !first.IsLast || first.PrevNode != "" || first.NextNode != "" {
		t.Fatalf("duplicate corrupted links: %+v", first)
	}
	depth, err := (&Pool{}).GetDepth(ctx, &api.DepthRequest{
		Symbol: node.Symbol, Transaction: api.DepthRequest_SELL, Count: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(depth.Data) != 1 || depth.Data[0].P != node.Price || depth.Data[0].V != node.Volume {
		t.Fatalf("expected best ask with original volume, got %v", depth)
	}
}
