package engine

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/streadway/amqp"
	"gome/api"
	"gome/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRabbitMQRedeliveryAndOutboxRecovery(t *testing.T) {
	port := os.Getenv("GOME_TEST_RABBITMQ_PORT")
	if port == "" {
		t.Skip("set GOME_TEST_RABBITMQ_PORT to an isolated RabbitMQ instance")
	}
	symbol := configureTestBook(t)
	Conf.MQconf = utils.RabbitMQ{Host: "127.0.0.1", Port: port, Username: "gome-test", Password: "gome-test"}
	request, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	conn, err := dialRabbitMQ()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	channel, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer channel.Close()
	defer func() {
		channel.QueueDelete(symbol, false, false, false)
		channel.QueueDelete("notice:"+symbol, false, false, false)
		publisher.Lock()
		defer publisher.Unlock()
		if publisher.conn != nil {
			publisher.conn.Close()
			publisher.conn = nil
		}
	}()

	maker := &api.OrderRequest{Uuid: "maker", Oid: "maker", Symbol: symbol, Transaction: api.TransactionType_SELL, Price: 1, Volume: 2}
	taker := &api.OrderRequest{Uuid: "taker", Oid: "taker", Symbol: symbol, Price: 1, Volume: 1}
	for _, order := range []*api.OrderRequest{maker, taker} {
		if _, err := (Order{}).DoOrder(request, order); err != nil {
			t.Fatal(err)
		}
	}

	response, err := (Order{}).DoOrder(request, taker)
	if err != nil || !response.GetDuplicate() {
		t.Fatalf("expected duplicate RPC response, got %v, %v", response, err)
	}
	conflict := &api.OrderRequest{Uuid: taker.Uuid, Oid: taker.Oid, Symbol: symbol, Price: taker.Price, Volume: 2}
	if _, err := (Order{}).DoOrder(request, conflict); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected conflict RPC response, got %v", err)
	}
	if queue, err := channel.QueueInspect(symbol); err != nil || queue.Messages != 2 {
		t.Fatalf("duplicate or conflict changed queue: %+v, %v", queue, err)
	}

	// Make the real broker accept a notice, then simulate a worker disconnect
	// before the confirmation is recorded in Redis or the order is ACKed.
	disconnected := errors.New("simulated worker disconnect")
	send := func(ctx context.Context, queue, id string, body []byte) error {
		if err := publishMessage(ctx, queue, id, body); err != nil {
			return err
		}
		return disconnected
	}
	if err := consumeOrders(request, symbol, send); !errors.Is(err, disconnected) {
		t.Fatalf("expected disconnect, got %v", err)
	}

	command, ok, err := channel.Get(symbol, false)
	if err != nil || !ok || !command.Redelivered || command.MessageId != taker.Oid || command.DeliveryMode != amqp.Persistent {
		t.Fatalf("expected durable redelivery of taker command, got %+v, %v", command, err)
	}
	if err := command.Nack(false, true); err != nil {
		t.Fatal(err)
	}

	workerCtx, stopWorker := context.WithCancel(request)
	stopped := make(chan error, 1)
	go func() {
		stopped <- consumeOrders(workerCtx, symbol, publishMessage)
	}()
	defer func() {
		stopWorker()
		if err := <-stopped; !errors.Is(err, context.Canceled) {
			t.Errorf("unexpected worker exit: %v", err)
		}
	}()

	receive := func() amqp.Delivery {
		t.Helper()
		for {
			message, ok, err := channel.Get("notice:"+symbol, true)
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				return message
			}
			select {
			case <-request.Done():
				t.Fatal("timed out waiting for recovered notice")
			case <-time.After(20 * time.Millisecond):
			}
		}
	}
	first, replay := receive(), receive()
	if first.DeliveryMode != amqp.Persistent || first.MessageId == "" ||
		first.MessageId != replay.MessageId || string(first.Body) != string(replay.Body) {
		t.Fatalf("notice replay changed identity or payload: %q, %q", first.MessageId, replay.MessageId)
	}
	checkOrderStatus(t, testOrder(t, maker), api.OrderStatus_ORDER_STATUS_PARTIALLY_FILLED, 1e8, 1e8, 0)
	checkOrderStatus(t, testOrder(t, taker), api.OrderStatus_ORDER_STATUS_FILLED, 1e8, 0, 0)
	if remaining := readFirst(t, testOrder(t, maker)).Volume; remaining != 1e8 {
		t.Fatalf("order redelivery matched twice: %v", remaining)
	}
}
