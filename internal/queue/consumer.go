package queue

import (
	"context"
	"encoding/json"
	"log"

	"github.com/rabbitmq/amqp091-go"
	"notificationservice/internal/db"
	"notificationservice/internal/model"
)

const (
	mainQueue  = "notifications"
	retryQueue = "notifications.retry"
	dlqQueue   = "notifications.dlq"
	dlx        = "notifications.dlx"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	store   *db.Store
}

func NewConsumer(url string, store *db.Store) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	c := &Consumer{conn: conn, channel: ch, store: store}
	if err := c.setup(); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Consumer) setup() error {
	if err := c.channel.ExchangeDeclare(dlx, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := c.channel.QueueDeclare(mainQueue, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    dlx,
		"x-dead-letter-routing-key": retryQueue,
	}); err != nil {
		return err
	}
	if _, err := c.channel.QueueDeclare(retryQueue, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": mainQueue,
		"x-message-ttl":             int32(10000),
	}); err != nil {
		return err
	}
	if _, err := c.channel.QueueDeclare(dlqQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := c.channel.QueueBind(dlqQueue, dlqQueue, dlx, false, nil); err != nil {
		return err
	}
	return nil
}

func (c *Consumer) Start(ctx context.Context) {
	msgs, err := c.channel.Consume(mainQueue, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("consume error: %v", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-msgs:
			c.handleMessage(m)
		}
	}
}

func (c *Consumer) handleMessage(m amqp.Delivery) {
	var in model.NotificationInput
	if err := json.Unmarshal(m.Body, &in); err != nil {
		log.Printf("invalid message: %v", err)
		m.Nack(false, false)
		return
	}
	if err := in.Validate(); err != nil {
		log.Printf("validation failed: %v", err)
		m.Ack(false)
		return
	}
	if err := c.store.CreateNotification(context.Background(), in); err != nil {
		attempts := attemptCount(m)
		if attempts >= 5 {
			// send to DLQ
			_ = c.channel.Publish("", dlqQueue, false, false, amqp.Publishing{Body: m.Body})
			m.Ack(false)
			return
		}
		m.Nack(false, false)
		return
	}
	log.Printf("stored notification from %s to %s", model.MaskEmail(in.Sender), model.MaskEmail(in.Recipient))
	m.Ack(false)
}

func attemptCount(m amqp.Delivery) int {
	v, ok := m.Headers["x-death"]
	if !ok {
		return 0
	}
	deaths, ok := v.([]interface{})
	if !ok || len(deaths) == 0 {
		return 0
	}
	table, ok := deaths[0].(amqp.Table)
	if !ok {
		return 0
	}
	if cnt, ok := table["count"].(int64); ok {
		return int(cnt)
	}
	return 0
}
