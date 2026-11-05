package engine

import (
	"context"
	"math"
	"testing"

	"github.com/stingbo/gome/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRejectInvalidOrdersBeforeIO(t *testing.T) {
	for _, test := range []struct {
		name          string
		symbol        string
		direction     api.TransactionType
		price, volume float64
	}{
		{"queue namespace", "notice:btc", api.TransactionType_BUY, 1, 1},
		{"unknown side", "btc", 9, 1, 1},
		{"negative price", "btc", api.TransactionType_BUY, -1, 1},
		{"zero volume", "btc", api.TransactionType_BUY, 1, 0},
		{"nonfinite volume", "btc", api.TransactionType_BUY, 1, math.NaN()},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := (Order{}).DoOrder(context.Background(), &api.OrderRequest{
				Uuid: "user", Oid: "order", Symbol: test.symbol, Transaction: test.direction, Price: test.price, Volume: test.volume,
			})
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %v", err)
			}
		})
	}
}
