package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"order-service/internal/domain/models"
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
		Brokers:         c.brokers,
		Topic:           c.topic,
		GroupID:         c.groupID,
		MinBytes:        10e3, // 10KB
		MaxBytes:        10e6, // 10MB
		MaxWait:         500 * time.Millisecond,
		ReadLagInterval: -1,
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

		msgCtx, msgCancel := context.WithTimeout(ctx, 5*time.Second)
		msg, err := c.reader.FetchMessage(msgCtx)
		msgCancel()

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}

			slog.Error("Error reading message from Kafka", "error", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Обработка сообщения
		slog.Info("Received message", "partition", msg.Partition, "offset", msg.Offset)

		var order models.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			slog.Error("Failed to parse message", "error", err, "raw_message", string(msg.Value))
			// Подтверждаем даже некорректные сообщения, чтобы не застревать
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				slog.Error("Failed to commit message", "error", err)
			}
			continue
		}

		// Проверяем, обрабатывали ли мы уже этот заказ
		if c.cache.Has(order.OrderUID) {
			slog.Info("Order already processed, skipping", "orderUID", order.OrderUID)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				slog.Error("Failed to commit message", "error", err)
			}
			continue
		}

		// Сохраняем заказ в базу данных
		saveCtx, saveCancel := context.WithTimeout(ctx, 5*time.Second)
		err = c.repo.SaveOrder(saveCtx, order)
		saveCancel()

		if err != nil {
			slog.Error("Failed to save order", "error", err, "orderUID", order.OrderUID)
			// Не подтверждаем сообщение, чтобы обработать его позже
			continue
		}

		// Сохраняем данные заказа в кеш
		cachedData, err := json.Marshal(order)
		if err != nil {
			slog.Error("Failed to marshal order for cache", "error", err)
		} else {
			c.cache.Set(order.OrderUID, cachedData)

			// Можно также сохранить сериализованные данные в БД для быстрого восстановления кеша
			if err := c.repo.CacheOrderData(order.OrderUID, cachedData); err != nil {
				slog.Error("Failed to cache order data in DB", "error", err)
			}
		}

		// Подтверждаем обработку сообщения
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("Failed to commit message", "error", err)
		} else {
			slog.Info("Message processed and committed", "orderUID", order.OrderUID)
		}

	}

}

func (c *OrderKafkaConsumer) Shutdown(ctx context.Context) error {
	if !c.isRunning || c.reader == nil {
		return nil
	}

	slog.Info("Kafka consumer shutting down")

	if err := c.reader.Close(); err != nil {
		slog.Error("Kafka reader close failed", "error", err)
		return err
	}

	c.isRunning = false
	return nil
}
