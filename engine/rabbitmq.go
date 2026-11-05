package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
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

	// Keep existing queue properties so deployments can upgrade with their
	// existing queues and workers.
	if _, err := channel.QueueDeclare(queue, false, false, false, false, nil); err != nil {
		return err
	}
	if err := channel.Confirm(false); err != nil {
		return err
	}
	confirmed := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	returned := channel.NotifyReturn(make(chan amqp.Return, 1))
	if err := channel.Publish("", queue, true, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
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
		case message := <-returned:
			return fmt.Errorf("RabbitMQ returned message for %s: %s", queue, message.ReplyText)
		default:
			return nil
		}
	case <-ctx.Done():
		return fmt.Errorf("wait for RabbitMQ confirmation: %w", ctx.Err())
	}
}

func ConsumeNewOrder(symbol string) error {
	return consume(symbol, func(body []byte) error {
		var order OrderNode
		if err := json.Unmarshal(body, &order); err != nil {
			return err
		}
		if order.Symbol != symbol || (order.Action != ADD && order.Action != DEL) {
			return fmt.Errorf("invalid order for queue %s", symbol)
		}
		DoOrder(order)
		return nil
	})
}

func ConsumeMatchOrder(symbol string) error {
	return consume("notice:"+symbol, func(body []byte) error {
		var result MatchResult
		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}
		log.Printf("order result: symbol=%s action=%d order=%s counterparty=%s volume=%g time=%s",
			symbol, result.Action, result.Node.Oid, result.MatchNode.Oid, result.MatchVolume, result.MatchTime)
		return nil
	})
}

func consume(queue string, handle func([]byte) error) error {
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
	if _, err := channel.QueueDeclare(queue, false, false, false, false, nil); err != nil {
		return err
	}
	messages, err := channel.Consume(queue, "", true, false, false, false, nil)
	if err != nil {
		return err
	}
	for message := range messages {
		if err := handle(message.Body); err != nil {
			log.Printf("consume %s: %v", queue, err)
		}
	}
	return fmt.Errorf("RabbitMQ consumer stopped for %s", queue)
}
