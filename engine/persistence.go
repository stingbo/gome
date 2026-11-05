package engine

import (
	"context"
	"fmt"
)

// The worker must not ACK broker deliveries against an ephemeral order book.
func checkRedisPersistence(ctx context.Context) error {
	for _, setting := range []struct{ key, required string }{
		{"appendonly", "yes"},
		{"appendfsync", "always"},
		{"no-appendfsync-on-rewrite", "no"},
		{"maxmemory-policy", "noeviction"},
	} {
		values, err := cache.ConfigGet(ctx, setting.key).Result()
		if err != nil {
			return fmt.Errorf("verify Redis %s: %w", setting.key, err)
		}
		if len(values) != 2 || values[1] != setting.required {
			return fmt.Errorf("Redis requires %s %s for order recovery", setting.key, setting.required)
		}
	}
	return nil
}
