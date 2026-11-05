package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
	"gome/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type orderTerms struct {
	User     string
	Side     int32
	Price    ticks
	Volume   ticks
	Accuracy int
}

type orderRecord struct {
	Terms     orderTerms
	Status    api.OrderStatus
	Filled    ticks
	Remaining ticks
	Cancelled ticks
}

func newOrderRecord(order OrderNode) orderRecord {
	return orderRecord{
		Terms: orderTerms{
			User: order.Uuid, Side: order.Transaction, Price: order.Price,
			Volume: order.Volume, Accuracy: order.Accuracy,
		},
		Status: api.OrderStatus_ORDER_STATUS_PENDING, Remaining: order.Volume,
	}
}

func orderStatusKey(symbol, id string) string {
	return symbol + ":order-status:" + id
}

// Keep the first request's normalized terms even if the publisher loses its
// confirmation. Existing legacy IDs without recoverable terms stay reserved.
var reserveOrder = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if current then
	return current
end
if redis.call("HEXISTS", KEYS[2], ARGV[1]) == 1 or
   redis.call("HEXISTS", KEYS[3], ARGV[2]) == 1 then
	return "legacy"
end
redis.call("SET", KEYS[1], ARGV[3])
return ""
`)

var markOrderQueued = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	redis.call("SET", KEYS[1], ARGV[2])
end
return 1
`)

func submitOrder(ctx context.Context, order *OrderNode, send func(context.Context, *OrderNode) error) (*api.OrderResponse, error) {
	record := newOrderRecord(*order)
	// The record contains only strings and integers.
	pending, _ := json.Marshal(record)
	key := orderStatusKey(order.Symbol, order.Oid)
	prior, err := reserveOrder.Run(ctx, cache,
		[]string{key, order.Symbol + ":accepted", order.NodeLink}, order.Oid, order.NodeName, pending).Text()
	if err != nil {
		log.Printf("reserve order for %s: %v", order.Symbol, err)
		return nil, status.Error(codes.Unavailable, "order status store unavailable")
	}
	duplicate := prior != ""
	expectedPending := string(pending)
	if duplicate {
		if prior == "legacy" {
			return nil, status.Error(codes.AlreadyExists, "order ID is already in use by a legacy order")
		}
		var existing orderRecord
		if err := json.Unmarshal([]byte(prior), &existing); err != nil {
			return nil, status.Error(codes.Internal, "invalid stored order status")
		}
		if existing.Terms != record.Terms {
			return nil, status.Error(codes.AlreadyExists, "order ID is already in use with different parameters")
		}
		if existing.Status != api.OrderStatus_ORDER_STATUS_PENDING {
			return duplicateOrderResponse(), nil
		}
		expectedPending = prior
	}

	if err := send(ctx, order); err != nil {
		log.Printf("publish order command for %s: %v", order.Symbol, err)
		return nil, status.Error(codes.Unavailable, "order queue unavailable")
	}
	record.Status = api.OrderStatus_ORDER_STATUS_QUEUED
	queued, _ := json.Marshal(record)
	// The worker may already have written a newer lifecycle state.
	if err := markOrderQueued.Run(ctx, cache, []string{key}, expectedPending, queued).Err(); err != nil {
		log.Printf("record queue confirmation for %s: %v", order.Symbol, err)
		return nil, status.Error(codes.Unavailable, "order status store unavailable")
	}
	if duplicate {
		return duplicateOrderResponse(), nil
	}
	return &api.OrderResponse{Message: "下单请求已入队"}, nil
}

func duplicateOrderResponse() *api.OrderResponse {
	return &api.OrderResponse{Message: "相同下单请求已受理", Duplicate: true}
}

func (o Order) GetOrderStatus(ctx context.Context, request *api.OrderStatusRequest) (*api.OrderStatusResponse, error) {
	if err := validateOrderIdentity(request.GetSymbol(), request.GetUuid(), request.GetOid()); err != nil {
		return nil, err
	}
	raw, err := cache.Get(ctx, orderStatusKey(request.Symbol, request.Oid)).Result()
	if err == redis.Nil {
		return nil, status.Error(codes.NotFound, "order status not found")
	}
	if err != nil {
		return nil, status.Error(codes.Unavailable, "order status store unavailable")
	}
	var record orderRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, status.Error(codes.Internal, "invalid stored order status")
	}
	if record.Terms.User != request.Uuid {
		return nil, status.Error(codes.NotFound, "order status not found")
	}
	return &api.OrderStatusResponse{
		Status: record.Status, Transaction: api.TransactionType(record.Terms.Side),
		PriceTicks: int64(record.Terms.Price), OriginalVolumeTicks: int64(record.Terms.Volume),
		FilledVolumeTicks: int64(record.Filled), RemainingVolumeTicks: int64(record.Remaining),
		CancelledVolumeTicks: int64(record.Cancelled), Accuracy: int32(record.Terms.Accuracy),
	}, nil
}

func readOrderRecord(ctx context.Context, tx *redis.Tx, key string) (*orderRecord, error) {
	raw, err := tx.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var record orderRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (record *orderRecord) fill(quantity ticks) error {
	if quantity <= 0 || quantity > record.Remaining {
		return fmt.Errorf("fill exceeds recorded remaining volume")
	}
	record.Filled += quantity
	record.Remaining -= quantity
	record.Status = api.OrderStatus_ORDER_STATUS_PARTIALLY_FILLED
	if record.Remaining == 0 {
		record.Status = api.OrderStatus_ORDER_STATUS_FILLED
	}
	return nil
}

// Build lifecycle updates before EXEC so an invalid record cannot leave
// status, book and outbox partially updated.
func (update *bookUpdate) orderRecords(ctx context.Context, tx *redis.Tx, order OrderNode) (map[string]orderRecord, error) {
	changes := make(map[string]orderRecord)
	key := orderStatusKey(order.Symbol, order.Oid)
	if order.Action == DEL {
		if len(update.events) == 0 {
			return changes, nil
		}
		record, err := readOrderRecord(ctx, tx, key)
		if err != nil || record == nil {
			return changes, err
		}
		if record.Remaining != update.events[0].Node.Volume {
			return nil, fmt.Errorf("cancel volume disagrees with order status for %s", order.Oid)
		}
		record.Status = api.OrderStatus_ORDER_STATUS_CANCELLED
		record.Cancelled = record.Remaining
		record.Remaining = 0
		changes[key] = *record
		return changes, nil
	}
	if len(update.levels) == 0 {
		// The order was already resting in a legacy book.
		return changes, nil
	}

	incoming := newOrderRecord(order)
	incoming.Status = api.OrderStatus_ORDER_STATUS_OPEN
	for _, event := range update.events {
		if err := incoming.fill(event.MatchVolume); err != nil {
			return nil, err
		}
		makerKey := orderStatusKey(order.Symbol, event.MatchNode.Oid)
		maker, err := readOrderRecord(ctx, tx, makerKey)
		if err != nil {
			return nil, err
		}
		// Historical book entries do not carry their original order size.
		if maker != nil {
			if err := maker.fill(event.MatchVolume); err != nil {
				return nil, fmt.Errorf("update maker %s: %w", event.MatchNode.Oid, err)
			}
			changes[makerKey] = *maker
		}
	}
	changes[key] = incoming
	return changes, nil
}
