package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"order-service/internal/models"
	"order-service/pkg/interfaces"

	"github.com/segmentio/kafka-go"
)

type OrderKafkaConsumer struct {
	brokers   []string
	topic     string
	groupID   string
	repo      interfaces.OrderRepository
	cache     interfaces.CacheRepository
	isRunning bool
	reader    *kafka.Reader
}

func NewOrderKafkaConsumer(brokers []string, topic, groupID string, repo interfaces.OrderRepository, cache interfaces.CacheRepository) *OrderKafkaConsumer {
	return &OrderKafkaConsumer{
		brokers: brokers,
		topic:   topic,
		groupID: groupID,
		repo:    repo,
		cache:   cache,
	}
}

func (c *OrderKafkaConsumer) Start(ctx context.Context) error {
	if c.isRunning {
		return nil
	}

	c.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.brokers,
		Topic:   c.topic,
		GroupID: c.groupID,
	})

	slog.Info("Kafka subscription started",
		"topic", c.topic,
		"brokers", c.brokers,
		"group_id", c.groupID)

	c.isRunning = true

	go c.processMessages(ctx)

	return nil
}

func (c *OrderKafkaConsumer) processMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("Kafka consumer context canceled")
			return
		default:
		}

		msgCtx, msgCancel := context.WithTimeout(ctx, 1*time.Second)
		msg, err := c.reader.ReadMessage(msgCtx)
		msgCancel()

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}

			slog.Error("Error reading message from Kafka", "error", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		msgValue := string(msg.Value)
		slog.Info("Received message", "kafka_message", msgValue, "partition", msg.Partition, "offset", msg.Offset)

		var order models.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			slog.Error("Failed to parse message", "error", err, "raw_message", msgValue)
			continue
		}
		slog.Info("Message parsed successfully", "order", order)

		if c.cache.Has(order.OrderUID) {
			slog.Info("Order already processed, skipping", "orderUID", order.OrderUID)
			continue
		}

		if err := c.repo.SaveOrder(order); err != nil {
			slog.Error("Failed to save order", "error", err, "orderUID", order.OrderUID)
			continue
		}

		c.cache.Set(order.OrderUID)
		slog.Info("Message processed", "orderUID", order.OrderUID)
	}
}

func (c *OrderKafkaConsumer) Shutdown(ctx context.Context) error {
	if !c.isRunning || c.reader == nil {
		return nil
	}

	slog.Info("Kafka consumer shutting down")

	if err := c.reader.Close(); err != nil {
		slog.Error("Kafka reader close failed", "error", err)
	}

	c.isRunning = false
	return nil
}
