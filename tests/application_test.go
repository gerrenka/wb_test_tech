package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"order-service/internal/config"
	appuse "order-service/internal/usecase"
	"order-service/mocks"
)

// Тест для метода Start приложения
func TestApplication_Start(t *testing.T) {
	// Создаем моки для интерфейсов
	mockRepo := new(mocks.OrderRepository)
	mockCache := new(mocks.CacheRepository)
	mockKafka := new(mocks.KafkaConsumer)
	mockHTTP := new(mocks.HTTPServer)

	// Настраиваем ожидания
	mockKafka.On("Start", mock.Anything).Return(nil)
	mockHTTP.On("Start").Return(nil)

	// Создаем экземпляр приложения с моками
	app := appuse.NewApplication(
		&config.Config{},
		mockRepo,
		mockCache,
		mockKafka,
		mockHTTP,
	)

	// Тестируем метод Start
	ctx := context.Background()
	err := app.Start(ctx)

	// Проверяем результаты
	assert.NoError(t, err)
	mockKafka.AssertExpectations(t)
	mockHTTP.AssertExpectations(t)
}

func TestApplication_Shutdown(t *testing.T) {
	// Создаем моки для интерфейсов
	mockRepo := new(mocks.OrderRepository)
	mockCache := new(mocks.CacheRepository)
	mockKafka := new(mocks.KafkaConsumer)
	mockHTTP := new(mocks.HTTPServer)

	// Настраиваем ожидания
	mockKafka.On("Shutdown", mock.Anything).Return(nil)
	mockHTTP.On("Shutdown", mock.Anything).Return(nil)

	// Создаем экземпляр приложения с моками
	app := appuse.NewApplication(
		&config.Config{},
		mockRepo,
		mockCache,
		mockKafka,
		mockHTTP,
	)

	// Тестируем метод Shutdown
	ctx := context.Background()
	err := app.Shutdown(ctx)

	// Проверяем результаты
	assert.NoError(t, err)
	mockKafka.AssertExpectations(t)
	mockHTTP.AssertExpectations(t)
}
