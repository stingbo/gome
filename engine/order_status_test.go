package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/stingbo/gome/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestOrderSubmissionRetryAndConflict(t *testing.T) {
	symbol := configureTestBook(t)
	request := &api.OrderRequest{Uuid: "u", Oid: "one", Symbol: symbol, Price: 1, Volume: 2}
	node := testOrder(t, request)
	calls := 0
	send := func(context.Context, *OrderNode) error {
		calls++
		if calls == 1 {
			return errors.New("publisher confirmation lost")
		}
		return nil
	}
	if _, err := submitOrder(ctx, node, send); status.Code(err) != codes.Unavailable {
		t.Fatalf("expected ambiguous publication, got %v", err)
	}
	checkOrderStatus(t, node, api.OrderStatus_ORDER_STATUS_PENDING, 0, 2e8, 0)

	conflict := *node
	conflict.Volume++
	if _, err := submitOrder(ctx, &conflict, send); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected conflict, got %v", err)
	}
	for i := 0; i < 2; i++ {
		response, err := submitOrder(ctx, node, send)
		if err != nil || !response.GetDuplicate() {
			t.Fatalf("expected successful duplicate response, got %v, %v", response, err)
		}
	}
	if calls != 2 {
		t.Fatalf("expected two publication attempts, got %d", calls)
	}
	checkOrderStatus(t, node, api.OrderStatus_ORDER_STATUS_QUEUED, 0, 2e8, 0)
	if _, err := (Order{}).GetOrderStatus(ctx, &api.OrderStatusRequest{Symbol: symbol, Oid: node.Oid, Uuid: "other"}); status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound for another user, got %v", err)
	}
}

func TestOrderStatusTracksBookTransactions(t *testing.T) {
	symbol := configureTestBook(t)
	maker := testOrder(t, &api.OrderRequest{Uuid: "maker", Oid: "maker", Symbol: symbol, Transaction: api.TransactionType_SELL, Price: 1, Volume: 3})
	taker := testOrder(t, &api.OrderRequest{Uuid: "taker", Oid: "taker", Symbol: symbol, Price: 1, Volume: 1})
	send := func(context.Context, *OrderNode) error { return nil }

	response, err := submitOrder(ctx, maker, send)
	if err != nil || response.GetDuplicate() {
		t.Fatalf("expected a new order, got %v, %v", response, err)
	}
	if err := applyOrder(ctx, *maker); err != nil {
		t.Fatal(err)
	}
	checkOrderStatus(t, maker, api.OrderStatus_ORDER_STATUS_OPEN, 0, 3e8, 0)

	// The worker may complete before the publisher confirmation is recorded.
	response, err = submitOrder(ctx, taker, func(ctx context.Context, order *OrderNode) error {
		return applyOrder(ctx, *order)
	})
	if err != nil || response.GetDuplicate() {
		t.Fatalf("expected new order, got %v, %v", response, err)
	}
	if err := applyOrder(ctx, *taker); err != nil {
		t.Fatal(err)
	}
	checkOrderStatus(t, taker, api.OrderStatus_ORDER_STATUS_FILLED, 1e8, 0, 0)
	checkOrderStatus(t, maker, api.OrderStatus_ORDER_STATUS_PARTIALLY_FILLED, 1e8, 2e8, 0)

	cancel := *maker
	cancel.Action, cancel.CommandID = DEL, "cancel"
	for i := 0; i < 2; i++ {
		if err := applyOrder(ctx, cancel); err != nil {
			t.Fatal(err)
		}
	}
	checkOrderStatus(t, maker, api.OrderStatus_ORDER_STATUS_CANCELLED, 1e8, 0, 2e8)
	response, err = submitOrder(ctx, maker, func(context.Context, *OrderNode) error {
		t.Fatal("completed duplicate must not be republished")
		return nil
	})
	if err != nil || !response.GetDuplicate() {
		t.Fatalf("expected terminal duplicate, got %v, %v", response, err)
	}
}

func checkOrderStatus(t *testing.T, order *OrderNode, want api.OrderStatus, filled, remaining, cancelled int64) {
	t.Helper()
	response, err := (Order{}).GetOrderStatus(ctx, &api.OrderStatusRequest{
		Symbol: order.Symbol, Uuid: order.Uuid, Oid: order.Oid,
	})
	if err != nil || response.Status != want || response.FilledVolumeTicks != filled ||
		response.RemainingVolumeTicks != remaining || response.CancelledVolumeTicks != cancelled ||
		response.OriginalVolumeTicks != int64(order.Volume) || response.PriceTicks != int64(order.Price) ||
		response.Transaction != api.TransactionType(order.Transaction) || response.Accuracy != int32(order.Accuracy) {
		t.Fatalf("expected %v, filled=%d remaining=%d cancelled=%d, got %v, %v", want, filled, remaining, cancelled, response, err)
	}
}

func TestFailedBookCommitKeepsQueuedStatus(t *testing.T) {
	symbol := configureTestBook(t)
	maker := testOrder(t, &api.OrderRequest{Uuid: "maker", Oid: "maker", Symbol: symbol, Transaction: api.TransactionType_SELL, Price: 1, Volume: 3})
	taker := testOrder(t, &api.OrderRequest{Uuid: "taker", Oid: "taker", Symbol: symbol, Price: 1, Volume: 1})
	if err := applyOrder(ctx, *maker); err != nil {
		t.Fatal(err)
	}
	if _, err := submitOrder(ctx, taker, func(context.Context, *OrderNode) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := cache.Set(ctx, outboxKey(symbol), "wrong Redis type", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := applyOrder(ctx, *taker); err == nil {
		t.Fatal("expected commit to fail")
	}
	checkOrderStatus(t, maker, api.OrderStatus_ORDER_STATUS_OPEN, 0, 3e8, 0)
	checkOrderStatus(t, taker, api.OrderStatus_ORDER_STATUS_QUEUED, 0, 1e8, 0)
}

func TestLegacyOrderIDRemainsReserved(t *testing.T) {
	symbol := configureTestBook(t)
	order := testOrder(t, &api.OrderRequest{Uuid: "user", Oid: "legacy", Symbol: symbol, Price: 1, Volume: 1})
	if err := cache.HSet(ctx, symbol+":accepted", order.Oid, 1).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := submitOrder(ctx, order, func(context.Context, *OrderNode) error {
		t.Fatal("legacy ID must not be republished")
		return nil
	}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected legacy ID conflict, got %v", err)
	}
}

func TestConcurrentConflictingSubmissionsReserveOneOrder(t *testing.T) {
	symbol := configureTestBook(t)
	order := testOrder(t, &api.OrderRequest{Uuid: "u", Oid: "race", Symbol: symbol, Price: 1, Volume: 2})
	conflict := *order
	conflict.Price++
	conflict.SetDepthHashKey()
	conflict.SetNodeLink()

	start := make(chan struct{})
	results := make(chan error, 2)
	published := make(chan ticks, 2)
	for _, request := range []*OrderNode{order, &conflict} {
		go func(request *OrderNode) {
			<-start
			_, err := submitOrder(ctx, request, func(_ context.Context, order *OrderNode) error {
				published <- order.Price
				return nil
			})
			results <- err
		}(request)
	}
	close(start)
	counts := make(map[codes.Code]int)
	for i := 0; i < 2; i++ {
		counts[status.Code(<-results)]++
	}
	if counts[codes.OK] != 1 || counts[codes.AlreadyExists] != 1 || len(published) != 1 {
		t.Fatalf("expected one publication and one conflict, got %v, publications=%d", counts, len(published))
	}
	winner := order
	if <-published == conflict.Price {
		winner = &conflict
	}
	checkOrderStatus(t, winner, api.OrderStatus_ORDER_STATUS_QUEUED, 0, 2e8, 0)
}
