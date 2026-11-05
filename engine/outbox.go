package engine

import (
	"context"
	"encoding/json"

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

func flushOutbox(ctx context.Context, symbol string, send func(context.Context, string, MatchResult) error) error {
	key := outboxKey(symbol)
	for {
		raw, err := cache.LIndex(ctx, key, 0).Result()
		if err == redis.Nil {
			return nil
		}
		if err != nil {
			return err
		}
		var event MatchResult
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			return err
		}
		if err := send(ctx, "notice:"+symbol, event); err != nil {
			return err
		}
		if err := removeConfirmed.Run(ctx, cache, []string{key}, raw).Err(); err != nil {
			return err
		}
	}
}
