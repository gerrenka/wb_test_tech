// pkg/interfaces/interfaces.go
package interfaces

import (
	"context"
	"order-service/internal/domain/models"
)

// CacheRepository представляет интерфейс для кеширования
type CacheRepository interface {
	Set(key string, data []byte)
	Get(key string) ([]byte, bool)
	Has(key string) bool
	Delete(key string)
	PrintContent()
}

// OrderRepository представляет интерфейс для работы с заказами в БД
type OrderRepository interface {
	SaveOrder(ctx context.Context, order models.Order) error
	GetOrder(ctx context.Context, orderUID string) (*models.Order, error)
	GetAllOrders() ([]string, error)
	CacheOrderData(orderUID string, orderData []byte) error
	GetCachedOrderData(orderUID string) ([]byte, error)
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
