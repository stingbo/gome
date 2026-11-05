package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stingbo/gome/api"
	"github.com/stingbo/gome/utils"
)

func configureTestBook(t *testing.T) string {
	t.Helper()
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
	return fmt.Sprintf("book-test-%d", time.Now().UnixNano())
}

func TestOrderBook(t *testing.T) {
	symbol := configureTestBook(t)
	add := func(id string, price float64) *OrderNode {
		node := testOrder(t, &api.OrderRequest{
			Uuid: "user", Oid: id, Symbol: symbol, Transaction: api.TransactionType_SELL, Price: price, Volume: 2,
		})
		if err := applyOrder(ctx, *node); err != nil {
			t.Fatal(err)
		}
		return node
	}
	node := add("one", 1.25)
	add("one", 1.25)
	add("two", 1.5)
	first := readFirst(t, node)
	if !first.IsFirst || !first.IsLast || first.PrevNode != "" || first.NextNode != "" {
		t.Fatalf("duplicate corrupted links: %+v", first)
	}
	depth, err := (&Pool{}).GetDepth(ctx, &api.DepthRequest{
		Symbol: node.Symbol, Transaction: api.DepthRequest_SELL, Count: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(depth.Data) != 1 || depth.Data[0].P != float64(node.Price) || depth.Data[0].V != float64(node.Volume) {
		t.Fatalf("expected best ask with original volume, got %v", depth)
	}
}

func readFirst(t *testing.T, node *OrderNode) OrderNode {
	t.Helper()
	first, err := cache.HGet(ctx, node.NodeLink, "f").Result()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := cache.HGet(ctx, node.NodeLink, first).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	var resting OrderNode
	if err := json.Unmarshal(raw, &resting); err != nil {
		t.Fatal(err)
	}
	return resting
}

func testOrder(t *testing.T, req *api.OrderRequest) *OrderNode {
	t.Helper()
	order, err := makeOrder(req, ADD, Conf.MeConf.Accuracy)
	if err != nil {
		t.Fatal(err)
	}
	return order
}
