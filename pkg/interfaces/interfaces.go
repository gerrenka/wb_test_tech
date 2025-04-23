package interfaces

import (
	"context"
	"order-service/internal/domain/models"
)

type SomeInterface interface {
	ProcessOrder(order models.Order) error
}

// CacheRepository представляет интерфейс для кеширования
type CacheRepository interface {
	Set(key string)
	Has(key string) bool
	Delete(key string)
	PrintContent()
}

// OrderRepository представляет интерфейс для работы с заказами в БД
type OrderRepository interface {
	SaveOrder(order models.Order) error
	GetOrder(orderUID string) (string, error)
	GetAllOrders() ([]string, error)
}

// KafkaConsumer представляет интерфейс для работы с Kafka
type KafkaConsumer interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// HTTPServer представляет интерфейс для HTTP-сервера
type HTTPServer interface {
	Start() error
	Shutdown(ctx context.Context) error
}
