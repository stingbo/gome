package engine

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
	"gome/api"
)

type levelUpdate struct {
	level  *priceLevel
	order  OrderNode
	volume decimal.Decimal
}

type bookUpdate struct {
	levels []*levelUpdate
	events []MatchResult
}

// applyOrder commits the book, replay marker and outbox in one Redis transaction.
func applyOrder(ctx context.Context, order OrderNode) error {
	marker := order.Symbol + ":accepted"
	id := order.Oid
	if order.Action == DEL {
		marker = order.Symbol + ":cancellations"
		id = order.CommandID
	}
	outbox := outboxKey(order.Symbol)
	keys := []string{marker, order.OrderDepthHashKey, order.OrderListSortSetKey, order.OrderListSortSetRKey, outbox}
	return cache.Watch(ctx, func(tx *redis.Tx) error {
		seen, err := tx.HExists(ctx, marker, id).Result()
		if err != nil || (id != "" && seen) {
			return err
		}
		var update bookUpdate
		switch order.Action {
		case ADD:
			err = update.add(ctx, tx, order)
		case DEL:
			err = update.cancel(ctx, tx, order)
		default:
			return fmt.Errorf("unknown order action %d", order.Action)
		}
		if err != nil {
			return err
		}
		if len(update.events) > 0 {
			// Redis does not roll back runtime type errors during EXEC.
			if err := tx.LLen(ctx, outbox).Err(); err != nil {
				return err
			}
		}
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			if id != "" {
				pipe.HSet(ctx, marker, id, 1)
			}
			for _, level := range update.levels {
				if err := level.write(ctx, pipe); err != nil {
					return err
				}
			}
			for _, event := range update.events {
				body, err := json.Marshal(event)
				if err != nil {
					return err
				}
				pipe.RPush(ctx, outbox, body)
			}
			return nil
		})
		return err
	}, keys...)
}

func (update *bookUpdate) add(ctx context.Context, tx *redis.Tx, order OrderNode) error {
	own, err := readLevel(ctx, tx, order)
	if err != nil {
		return err
	}
	// Resting orders created before the accepted-order index still deduplicate.
	existing, err := own.level.get(ctx, tx, order.NodeName)
	if err != nil || existing != nil {
		return err
	}

	price := strconv.FormatFloat(order.Price, 'f', -1, 64)
	var prices []string
	if order.Transaction == int32(api.TransactionType_SELL) {
		prices, err = tx.ZRevRangeByScore(ctx, order.OrderListSortSetRKey, &redis.ZRangeBy{Min: price, Max: "+inf"}).Result()
	} else {
		prices, err = tx.ZRangeByScore(ctx, order.OrderListSortSetRKey, &redis.ZRangeBy{Min: "-inf", Max: price}).Result()
	}
	if err != nil {
		return err
	}
	for _, price := range prices {
		maker := order
		maker.Price, err = strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}
		maker.Transaction = 1 - order.Transaction
		maker.SetListSortSetKey()
		maker.SetDepthHashKey()
		maker.SetNodeLink()

		level, err := readLevel(ctx, tx, maker)
		if err != nil {
			return err
		}
		if level.level.first == "" {
			return fmt.Errorf("missing FIFO for price level %s", level.level.key)
		}
		if err := update.matchLevel(ctx, tx, &order, level); err != nil {
			return err
		}
		update.levels = append(update.levels, level)
		if order.Volume == 0 {
			return nil
		}
	}

	// An exhausted opposite level at the incoming price has the same Redis
	// link and depth keys. Its deletion is staged before the new level.
	if len(update.levels) > 0 {
		last := update.levels[len(update.levels)-1]
		if last.level.key == own.level.key {
			own.level = &priceLevel{key: last.level.key, changes: make(map[string]*OrderNode)}
			own.volume = decimal.Zero
		}
	}
	if err := own.level.append(ctx, tx, order); err != nil {
		return err
	}
	own.volume = own.volume.Add(decimal.NewFromFloat(order.Volume))
	update.levels = append(update.levels, own)
	return nil
}

func (update *bookUpdate) matchLevel(ctx context.Context, tx *redis.Tx, order *OrderNode, level *levelUpdate) error {
	for order.Volume > 0 && level.level.first != "" {
		maker, err := level.level.get(ctx, tx, level.level.first)
		if err != nil {
			return err
		}
		if maker == nil || maker.Transaction == order.Transaction || maker.PrevNode != "" {
			return fmt.Errorf("invalid maker at %s", level.level.key)
		}
		remaining := decimal.NewFromFloat(order.Volume)
		available := decimal.NewFromFloat(maker.Volume)
		filled := decimal.Min(remaining, available)
		volume, _ := filled.Float64()
		order.Volume, _ = remaining.Sub(filled).Float64()
		level.volume = level.volume.Sub(filled)
		if filled.Equal(available) {
			if err := level.level.remove(ctx, tx, maker); err != nil {
				return err
			}
		} else {
			maker.Volume, _ = available.Sub(filled).Float64()
			level.level.changes[maker.NodeName] = maker
		}
		update.event(ADD, *order, *maker, volume)
	}
	return nil
}

func (update *bookUpdate) cancel(ctx context.Context, tx *redis.Tx, order OrderNode) error {
	level, err := readLevel(ctx, tx, order)
	if err != nil {
		return err
	}
	resting, err := level.level.get(ctx, tx, order.NodeName)
	if err != nil || resting == nil {
		return err
	}
	if resting.Uuid != order.Uuid || resting.Transaction != order.Transaction {
		return nil
	}
	if err := level.level.remove(ctx, tx, resting); err != nil {
		return err
	}
	order.Volume = resting.Volume
	level.volume = level.volume.Sub(decimal.NewFromFloat(resting.Volume))
	update.levels = append(update.levels, level)
	update.event(DEL, order, order, 0)
	return nil
}

func (update *bookUpdate) event(action int8, order, maker OrderNode, volume float64) {
	identity, _ := json.Marshal([4]string{order.Symbol, strconv.Itoa(int(action)), order.Oid, maker.Oid})
	update.events = append(update.events, MatchResult{
		EventID: fmt.Sprintf("%x", sha256.Sum256(identity)),
		Action:  action, Node: order, MatchNode: maker, MatchVolume: volume,
		MatchTime: time.Now().Format(timeFormat),
	})
}

func readLevel(ctx context.Context, tx *redis.Tx, order OrderNode) (*levelUpdate, error) {
	level, err := loadLevel(ctx, tx, order.NodeLink)
	if err != nil {
		return nil, err
	}
	raw, err := tx.HGet(ctx, order.OrderDepthHashKey, order.OrderDepthHashField).Result()
	volume := decimal.Zero
	if err != redis.Nil {
		if err != nil {
			return nil, err
		}
		volume, err = decimal.NewFromString(raw)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.ZScore(ctx, order.OrderListSortSetKey, strconv.FormatFloat(order.Price, 'f', -1, 64)).Err(); err != nil && err != redis.Nil {
		return nil, err
	}
	return &levelUpdate{level: level, order: order, volume: volume}, nil
}

func (update *levelUpdate) write(ctx context.Context, pipe redis.Pipeliner) error {
	if err := update.level.write(ctx, pipe); err != nil {
		return err
	}
	order := update.order
	if update.level.first == "" {
		pipe.HDel(ctx, order.OrderDepthHashKey, order.OrderDepthHashField)
		pipe.ZRem(ctx, order.OrderListSortSetKey, order.Price)
		return nil
	}
	if !update.volume.IsPositive() {
		return fmt.Errorf("nonpositive depth at %s", update.level.key)
	}
	pipe.HSet(ctx, order.OrderDepthHashKey, order.OrderDepthHashField, update.volume.String())
	pipe.ZAdd(ctx, order.OrderListSortSetKey, &redis.Z{Score: order.Price, Member: order.Price})
	return nil
}
