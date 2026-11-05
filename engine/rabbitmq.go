package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

var publisher struct {
	sync.Mutex
	conn *amqp.Connection
}

func dialRabbitMQ() (*amqp.Connection, error) {
	config := Conf.MQconf
	address := url.URL{
		Scheme: "amqp",
		User:   url.UserPassword(config.Username, config.Password),
		Host:   net.JoinHostPort(config.Host, config.Port),
		Path:   "/",
	}
	conn, err := amqp.DialConfig(address.String(), amqp.Config{Dial: amqp.DefaultDial(5 * time.Second)})
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}
	return conn, nil
}

func publisherChannel() (*amqp.Channel, error) {
	publisher.Lock()
	defer publisher.Unlock()

	if publisher.conn == nil || publisher.conn.IsClosed() {
		conn, err := dialRabbitMQ()
		if err != nil {
			return nil, err
		}
		publisher.conn = conn
	}
	return publisher.conn.Channel()
}

func publish(ctx context.Context, queue string, value interface{}) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	channel, err := publisherChannel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := declareQueue(channel, queue); err != nil {
		return err
	}
	if err := channel.Confirm(false); err != nil {
		return err
	}
	confirmed := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	returned := channel.NotifyReturn(make(chan amqp.Return, 1))
	if err := channel.Publish("", queue, true, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		MessageId:    messageID(value),
	}); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	select {
	case confirmation, open := <-confirmed:
		if !open || !confirmation.Ack {
			return fmt.Errorf("RabbitMQ did not confirm message for %s", queue)
		}
		// An unroutable mandatory publish is returned before it is confirmed.
		select {
		case message, open := <-returned:
			if open {
				return fmt.Errorf("RabbitMQ returned message for %s: %s", queue, message.ReplyText)
			}
		default:
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for RabbitMQ confirmation: %w", ctx.Err())
	}
}

func messageID(value interface{}) string {
	switch message := value.(type) {
	case MatchResult:
		return message.EventID
	case *OrderNode:
		if message.Action == DEL {
			return message.CommandID
		}
		return message.Oid
	default:
		return ""
	}
}

func declareQueue(channel *amqp.Channel, queue string) error {
	_, err := channel.QueueDeclare(queue, true, false, false, false, nil)
	return err
}

func ConsumeNewOrder(ctx context.Context, symbol string) error {
	if !symbolPattern.MatchString(symbol) {
		return fmt.Errorf("invalid symbol")
	}
	for {
		err := consumeOrders(ctx, symbol, func(ctx context.Context, queue string, result MatchResult) error {
			return publish(ctx, queue, result)
		})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Printf("order worker %s: %v; retrying", symbol, err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func consumeOrders(ctx context.Context, symbol string, send func(context.Context, string, MatchResult) error) error {
	if err := checkRedisPersistence(ctx); err != nil {
		return err
	}
	return consume(ctx, symbol, true, func() error {
		return flushOutbox(ctx, symbol, send)
	}, func(delivery amqp.Delivery) error {
		order, err := decodeOrder(symbol, delivery.Body)
		if err != nil {
			log.Printf("reject order in %s: %v", symbol, err)
			return delivery.Reject(false)
		}
		if err := applyOrder(ctx, order); err != nil {
			return err
		}
		if err := flushOutbox(ctx, symbol, send); err != nil {
			return err
		}
		return delivery.Ack(false)
	})
}

func decodeOrder(symbol string, body []byte) (OrderNode, error) {
	var order OrderNode
	if err := json.Unmarshal(body, &order); err != nil {
		return order, err
	}
	if order.Symbol != symbol || (order.Action != ADD && order.Action != DEL) ||
		(order.Transaction != 0 && order.Transaction != 1) ||
		strings.TrimSpace(order.Uuid) == "" || strings.TrimSpace(order.Oid) == "" ||
		strings.Contains(order.Uuid, ":") || strings.Contains(order.Oid, ":") ||
		!(order.Price > 0) || math.IsInf(order.Price, 0) ||
		(order.Action == ADD && !(order.Volume > 0)) {
		return order, fmt.Errorf("invalid order for queue %s", symbol)
	}
	order.SetListSortSetKey()
	order.SetDepthHashKey()
	order.SetNodeName()
	order.SetNodeLink()
	return order, nil
}

func ConsumeMatchOrder(symbol string) error {
	return consume(ctx, "notice:"+symbol, false, nil, func(delivery amqp.Delivery) error {
		var result MatchResult
		if err := json.Unmarshal(delivery.Body, &result); err != nil {
			log.Printf("reject notice for %s: %v", symbol, err)
			return delivery.Reject(false)
		}
		log.Printf("order result: id=%s symbol=%s action=%d order=%s counterparty=%s volume=%g time=%s",
			result.EventID, symbol, result.Action, result.Node.Oid, result.MatchNode.Oid, result.MatchVolume, result.MatchTime)
		return delivery.Ack(false)
	})
}

func consume(ctx context.Context, queue string, exclusive bool, start func() error, handle func(amqp.Delivery) error) error {
	conn, err := dialRabbitMQ()
	if err != nil {
		return err
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()
	if err := declareQueue(channel, queue); err != nil {
		return err
	}
	if err := channel.Qos(1, 0, false); err != nil {
		return err
	}
	messages, err := channel.Consume(queue, "", false, exclusive, false, false, nil)
	if err != nil {
		return err
	}
	if start != nil {
		if err := start(); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message, open := <-messages:
			if !open {
				return fmt.Errorf("RabbitMQ consumer stopped for %s", queue)
			}
			if err := handle(message); err != nil {
				// Closing the channel requeues the unacknowledged delivery.
				return fmt.Errorf("consume %s: %w", queue, err)
			}
		}
	}
}
