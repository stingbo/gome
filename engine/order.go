package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/stingbo/gome/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"regexp"
	"strings"
)

type Order struct{}

func (o Order) DoOrder(ctx context.Context, request *api.OrderRequest) (*api.OrderResponse, error) {
	orderNode, err := makeOrder(request, ADD, Conf.MeConf.Accuracy)
	if err != nil {
		return nil, err
	}
	return submitOrder(ctx, orderNode, publishOrder)
}

func (o Order) DeleteOrder(ctx context.Context, request *api.OrderRequest) (*api.OrderResponse, error) {
	orderNode, err := makeOrder(request, DEL, Conf.MeConf.Accuracy)
	if err != nil {
		return nil, err
	}
	var commandID [16]byte
	if _, err := rand.Read(commandID[:]); err != nil {
		return nil, status.Error(codes.Internal, "cannot generate cancellation ID")
	}
	orderNode.CommandID = hex.EncodeToString(commandID[:])

	// 删除队列
	if err := publishOrder(ctx, orderNode); err != nil {
		log.Printf("publish order command for %s: %v", orderNode.Symbol, err)
		return nil, status.Error(codes.Unavailable, "order queue unavailable")
	}
	response := &api.OrderResponse{Message: "撤单请求已入队"}

	return response, nil
}

var symbolPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

func validateOrderIdentity(symbol, user, order string) error {
	if !symbolPattern.MatchString(symbol) {
		return status.Error(codes.InvalidArgument, "invalid symbol")
	}
	for _, id := range []string{user, order} {
		if strings.TrimSpace(id) == "" || len(id) > 128 || strings.Contains(id, ":") {
			return status.Error(codes.InvalidArgument, "invalid user or order ID")
		}
	}
	return nil
}

func makeOrder(request *api.OrderRequest, action int8, accuracy int) (*OrderNode, error) {
	if err := validateOrderIdentity(request.GetSymbol(), request.GetUuid(), request.GetOid()); err != nil {
		return nil, err
	}
	if request.Transaction != api.TransactionType_BUY && request.Transaction != api.TransactionType_SELL {
		return nil, status.Error(codes.InvalidArgument, "invalid order side")
	}
	price, err := toOrderTicks(request.Price, accuracy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "price must be positive and a whole tick within range")
	}
	var volume ticks
	if action == ADD {
		volume, err = toOrderTicks(request.Volume, accuracy)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "volume must be positive and a whole tick within range")
		}
	}
	return newOrderNode(request, action, accuracy, price, volume), nil
}
