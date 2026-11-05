package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

func outboxKey(symbol string) string {
	return symbol + ":outbox"
}

// The head is removed only after broker confirmation. Comparing the payload
// prevents a worker losing its consumer connection from removing the next event.
var removeConfirmed = redis.NewScript(`
if redis.call("LINDEX", KEYS[1], 0) == ARGV[1] then
	return redis.call("LPOP", KEYS[1])
end
return ""
`)

type noticeSender func(context.Context, string, string, []byte) error

func flushOutbox(ctx context.Context, symbol string, send noticeSender) error {
	key := outboxKey(symbol)
	for {
		raw, err := cache.LIndex(ctx, key, 0).Result()
		if err == redis.Nil {
			return nil
		}
		if err != nil {
			return err
		}
		body := []byte(raw)
		var event struct{ EventID string }
		if err := json.Unmarshal(body, &event); err != nil {
			return err
		}
		if event.EventID == "" {
			return fmt.Errorf("missing event ID in outbox %s", key)
		}
		// Forward historical notices unchanged, even if they contain fractional ticks.
		if err := send(ctx, "notice:"+symbol, event.EventID, body); err != nil {
			return err
		}
		if err := removeConfirmed.Run(ctx, cache, []string{key}, raw).Err(); err != nil {
			return err
		}
	}
}
