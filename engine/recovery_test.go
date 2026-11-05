package engine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gome/api"
)

func TestRecoverOrderAndOutbox(t *testing.T) {
	symbol := configureTestBook(t)
	command := func(id string, side api.TransactionType, volume float64) OrderNode {
		node := testOrder(t, &api.OrderRequest{Uuid: "user", Oid: id, Symbol: symbol, Transaction: side, Price: 1, Volume: volume})
		return *node
	}
	maker := command("maker", api.TransactionType_SELL, 3)
	taker := command("taker", api.TransactionType_BUY, 1)
	if err := applyOrder(ctx, maker); err != nil {
		t.Fatal(err)
	}

	// A stopped Redis connection before commit must leave the order replayable.
	stopped, cancel := context.WithCancel(ctx)
	cancel()
	if err := applyOrder(stopped, taker); err == nil {
		t.Fatal("expected interrupted Redis operation")
	}
	if err := applyOrder(ctx, taker); err != nil {
		t.Fatal(err)
	}
	if err := applyOrder(ctx, taker); err != nil {
		t.Fatal(err)
	}

	// The broker may accept a publish even if the worker loses its confirmation.
	var delivered []MatchResult
	lostConfirmation := errors.New("lost confirmation")
	send := func(_ context.Context, _, _ string, body []byte) error {
		result := decodeTestNotice(t, body)
		delivered = append(delivered, result)
		if len(delivered) == 1 {
			return lostConfirmation
		}
		return nil
	}
	if err := flushOutbox(ctx, symbol, send); !errors.Is(err, lostConfirmation) {
		t.Fatalf("expected lost confirmation, got %v", err)
	}
	if err := flushOutbox(ctx, symbol, send); err != nil {
		t.Fatal(err)
	}
	if err := flushOutbox(ctx, symbol, send); err != nil {
		t.Fatal(err)
	}
	if len(delivered) != 2 || delivered[0].EventID == "" || delivered[0].EventID != delivered[1].EventID {
		t.Fatalf("expected same event on retry, got %+v", delivered)
	}
	resting := readFirst(t, &maker)
	if resting.Volume != 2e8 {
		t.Fatalf("replay changed book volume: %v", resting.Volume)
	}
}

func TestOutboxWriteFailureDoesNotChangeBook(t *testing.T) {
	symbol := configureTestBook(t)
	maker := testOrder(t, &api.OrderRequest{Uuid: "user", Oid: "maker", Symbol: symbol, Transaction: api.TransactionType_SELL, Price: 1, Volume: 2})
	taker := testOrder(t, &api.OrderRequest{Uuid: "user", Oid: "taker", Symbol: symbol, Price: 1, Volume: 1})
	if err := applyOrder(ctx, *maker); err != nil {
		t.Fatal(err)
	}
	if err := cache.Set(ctx, outboxKey(symbol), "wrong Redis type", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := applyOrder(ctx, *taker); err == nil {
		t.Fatal("expected outbox write to be rejected")
	}
	if volume := readFirst(t, maker).Volume; volume != maker.Volume {
		t.Fatalf("failed transaction changed volume to %v", volume)
	}
	if accepted, err := cache.HExists(ctx, symbol+":accepted", taker.Oid).Result(); err != nil || accepted {
		t.Fatalf("failed transaction marked order accepted: %v, %v", accepted, err)
	}
	if err := cache.Del(ctx, outboxKey(symbol)).Err(); err != nil {
		t.Fatal(err)
	}
	if err := applyOrder(ctx, *taker); err != nil {
		t.Fatal(err)
	}
	if volume := readFirst(t, maker).Volume; volume != 1e8 {
		t.Fatalf("replayed order failed to match: %v", volume)
	}
}

func TestMatchFIFOThenRestOnOppositeSideAtSamePrice(t *testing.T) {
	symbol := configureTestBook(t)
	for _, order := range []struct {
		id     string
		side   api.TransactionType
		volume float64
	}{
		{"first", api.TransactionType_SELL, 1},
		{"second", api.TransactionType_SELL, 2},
		{"buy", api.TransactionType_BUY, 4},
	} {
		node := testOrder(t, &api.OrderRequest{Uuid: "user", Oid: order.id, Symbol: symbol, Transaction: order.side, Price: 1, Volume: order.volume})
		if err := applyOrder(ctx, *node); err != nil {
			t.Fatal(err)
		}
	}
	var makers []string
	err := flushOutbox(ctx, symbol, func(_ context.Context, _, _ string, body []byte) error {
		event := decodeTestNotice(t, body)
		makers = append(makers, event.MatchNode.Oid)
		return nil
	})
	if err != nil || len(makers) != 2 || makers[0] != "first" || makers[1] != "second" {
		t.Fatalf("expected FIFO matches, got %v, %v", makers, err)
	}
	for _, side := range []api.DepthRequest_DepthType{api.DepthRequest_BUY, api.DepthRequest_SELL} {
		depth, err := (&Pool{}).GetDepth(ctx, &api.DepthRequest{Symbol: symbol, Transaction: side})
		if err != nil {
			t.Fatal(err)
		}
		if side == api.DepthRequest_BUY && (len(depth.Data) != 1 || depth.Data[0].V != 1e8) {
			t.Fatalf("wrong buy depth: %v", depth)
		}
		if side == api.DepthRequest_SELL && depth.Total != 0 {
			t.Fatalf("wrong sell depth: %v", depth)
		}
	}
}

func TestRedeliveredCancellationDoesNotChangeItsOriginalOutcome(t *testing.T) {
	symbol := configureTestBook(t)
	order := testOrder(t, &api.OrderRequest{Uuid: "user", Oid: "order", Symbol: symbol, Price: 1, Volume: 2})
	cancel := *order
	cancel.Action = DEL
	cancel.CommandID = "first-cancel"
	for _, command := range []OrderNode{cancel, *order, cancel} {
		if err := applyOrder(ctx, command); err != nil {
			t.Fatal(err)
		}
	}
	if resting := readFirst(t, order); resting.Volume != order.Volume {
		t.Fatalf("redelivery unexpectedly cancelled order: %+v", resting)
	}
	cancel.CommandID = "new-cancel"
	for i := 0; i < 2; i++ {
		if err := applyOrder(ctx, cancel); err != nil {
			t.Fatal(err)
		}
	}
	var events []MatchResult
	if err := flushOutbox(ctx, symbol, func(_ context.Context, _, _ string, body []byte) error {
		event := decodeTestNotice(t, body)
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != DEL || events[0].Node.Volume != order.Volume {
		t.Fatalf("expected one cancellation event with remaining volume, got %+v", events)
	}
}

func decodeTestNotice(t *testing.T, body []byte) MatchResult {
	t.Helper()
	var event MatchResult
	if err := json.Unmarshal(body, &event); err != nil {
		t.Fatal(err)
	}
	return event
}
