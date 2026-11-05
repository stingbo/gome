package engine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stingbo/gome/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMakeOrderRejectsSubTickValues(t *testing.T) {
	for _, req := range []*api.OrderRequest{
		{Uuid: "user", Oid: "order", Symbol: "btc", Price: 1e-9, Volume: 1},
		{Uuid: "user", Oid: "order", Symbol: "btc", Price: 1, Volume: 1.1e-8},
	} {
		if _, err := makeOrder(req, ADD, 8); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument for %v, got %v", req, err)
		}
	}
}

func TestLegacyIntegralJSON(t *testing.T) {
	var order OrderNode
	if err := json.Unmarshal([]byte(`{"Price":1.2345e8,"Volume":30000000.0}`), &order); err != nil {
		t.Fatal(err)
	}
	if order.Price != 123450000 || order.Volume != 30000000 {
		t.Fatalf("legacy order values changed: %+v", order)
	}
	if err := json.Unmarshal([]byte(`{"Price":1,"Volume":0.5}`), &order); err == nil {
		t.Fatal("fractional legacy ticks must not be silently rounded")
	}
}

func TestExactMatchingAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name     string
		commands []float64
		cancel   bool
		want     []int64
	}{
		{"partial then cancel", []float64{0.1}, true, []int64{10000000, 0}},
		{"partial then complete", []float64{0.1, 0.2}, false, []int64{10000000, 20000000}},
		{"complete in one fill", []float64{0.3}, false, []int64{30000000}},
	} {
		t.Run(test.name, func(t *testing.T) {
			symbol := configureTestBook(t)
			maker := testOrder(t, &api.OrderRequest{
				Uuid: "maker", Oid: "maker", Symbol: symbol, Transaction: api.TransactionType_SELL, Price: 0.12345678, Volume: 0.3,
			})
			if err := applyOrder(ctx, *maker); err != nil {
				t.Fatal(err)
			}
			for i, volume := range test.commands {
				taker := testOrder(t, &api.OrderRequest{
					Uuid: "taker", Oid: string(rune('a' + i)), Symbol: symbol, Price: 0.12345678, Volume: volume,
				})
				if err := applyOrder(ctx, *taker); err != nil {
					t.Fatal(err)
				}
				if i == 0 && volume == 0.1 {
					if got := readFirst(t, maker).Volume; got != 20000000 {
						t.Fatalf("partial fill left %d ticks", got)
					}
				}
			}
			if test.cancel {
				cancel := *maker
				cancel.Action, cancel.CommandID = DEL, "cancel-maker"
				if err := applyOrder(ctx, cancel); err != nil {
					t.Fatal(err)
				}
			}
			var events []MatchResult
			if err := flushOutbox(ctx, symbol, func(_ context.Context, _, _ string, body []byte) error {
				result := decodeTestNotice(t, body)
				events = append(events, result)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if len(events) != len(test.want) {
				t.Fatalf("expected %d events, got %d", len(test.want), len(events))
			}
			for i, event := range events {
				if int64(event.MatchVolume) != test.want[i] || event.Node.Price != 12345678 {
					t.Fatalf("unexpected event: %+v", event)
				}
			}
			if test.cancel && (events[len(events)-1].Action != DEL || events[len(events)-1].Node.Volume != 20000000) {
				t.Fatalf("wrong cancellation: %+v", events[len(events)-1])
			}
			for _, side := range []api.DepthRequest_DepthType{api.DepthRequest_BUY, api.DepthRequest_SELL} {
				depth, err := (&Pool{}).GetDepth(ctx, &api.DepthRequest{Symbol: symbol, Transaction: side})
				if err != nil || depth.Total != 0 || len(depth.Data) != 0 {
					t.Fatalf("expected empty book on %v, got %v, %v", side, depth, err)
				}
			}
		})
	}
}

func TestMatchLegacyIntegralOrderInRedis(t *testing.T) {
	symbol := configureTestBook(t)
	maker := testOrder(t, &api.OrderRequest{
		Uuid: "maker", Oid: "maker", Symbol: symbol, Transaction: api.TransactionType_SELL, Price: 0.12345678, Volume: 0.3,
	})
	if err := applyOrder(ctx, *maker); err != nil {
		t.Fatal(err)
	}
	raw, err := cache.HGet(ctx, maker.NodeLink, maker.NodeName).Result()
	if err != nil {
		t.Fatal(err)
	}
	raw = strings.Replace(raw, `"Price":12345678`, `"Price":1.2345678e7`, 1)
	raw = strings.Replace(raw, `"Volume":30000000`, `"Volume":30000000.0`, 1)
	if err := cache.HSet(ctx, maker.NodeLink, maker.NodeName, raw).Err(); err != nil {
		t.Fatal(err)
	}
	if err := cache.HSet(ctx, maker.OrderDepthHashKey, maker.OrderDepthHashField, "30000000.0").Err(); err != nil {
		t.Fatal(err)
	}
	taker := testOrder(t, &api.OrderRequest{
		Uuid: "taker", Oid: "taker", Symbol: symbol, Price: 0.12345678, Volume: 0.1,
	})
	if err := applyOrder(ctx, *taker); err != nil {
		t.Fatal(err)
	}
	if remaining := readFirst(t, maker).Volume; remaining != 20000000 {
		t.Fatalf("expected 20000000 ticks remaining, got %d", remaining)
	}
}

func TestLegacyNoticeIsReplayedWithoutConvertingFractionalTicks(t *testing.T) {
	symbol := configureTestBook(t)
	body := `{"EventID":"legacy","MatchVolume":0.125,"Node":{"Price":100000000,"Volume":0.375}}`
	if err := cache.RPush(ctx, outboxKey(symbol), body).Err(); err != nil {
		t.Fatal(err)
	}
	var sent bool
	err := flushOutbox(ctx, symbol, func(_ context.Context, queue, id string, raw []byte) error {
		sent = true
		if queue != "notice:"+symbol || id != "legacy" || string(raw) != body {
			t.Fatalf("unexpected notice: %q %q %s", queue, id, raw)
		}
		return nil
	})
	if err != nil || !sent {
		t.Fatalf("legacy notice was not sent: %v", err)
	}
}

func TestDepthTickParsingPreservesLargeIntegers(t *testing.T) {
	const total = "9223372036854775809"
	volume, err := parseDepthTicks(total + ".0")
	if err != nil || volume.String() != total {
		t.Fatalf("expected exact aggregate %s, got %v, %v", total, volume, err)
	}
}
