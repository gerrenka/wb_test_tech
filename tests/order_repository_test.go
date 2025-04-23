package main

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestPostgresRepository_SaveOrder(t *testing.T) {
	// Создаем мок для базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Ошибка при создании мока БД: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close db", "error", err)
		}
	}()
	

	// Создаем репозиторий с моком БД
	repo := NewPostgresRepository(db)

	// Тестовый заказ
	order := Order{OrderUID: "test-order-123"}

	// Настраиваем ожидание запроса
	mock.ExpectExec("INSERT INTO orders").
		WithArgs(order.OrderUID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Вызываем тестируемый метод
	err = repo.SaveOrder(order)

	// Проверяем результаты
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())

	// Тестируем случай с ошибкой
	expectedError := errors.New("database error")
	mock.ExpectExec("INSERT INTO orders").
		WithArgs("error-order").
		WillReturnError(expectedError)

	// Вызываем тестируемый метод с ошибкой
	err = repo.SaveOrder(Order{OrderUID: "error-order"})

	// Проверяем результаты
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestPostgresRepository_GetOrder(t *testing.T) {
	// Создаем мок для базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Ошибка при создании мока БД: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close db", "error", err)
		}
	}()
	

	// Создаем репозиторий с моком БД
	repo := NewPostgresRepository(db)

	// Тестовый ID заказа
	orderUID := "test-order-456"

	// Настраиваем ожидание запроса
	rows := sqlmock.NewRows([]string{"order_uid"}).
		AddRow(orderUID)
	mock.ExpectQuery("SELECT order_uid FROM orders WHERE order_uid = \\$1").
		WithArgs(orderUID).
		WillReturnRows(rows)

	// Вызываем тестируемый метод
	foundUID, err := repo.GetOrder(orderUID)

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Equal(t, orderUID, foundUID)
	assert.NoError(t, mock.ExpectationsWereMet())

	// Тестируем случай, когда заказ не найден
	mock.ExpectQuery("SELECT order_uid FROM orders WHERE order_uid = \\$1").
		WithArgs("not-found").
		WillReturnError(sqlmock.ErrNoRows)

	// Вызываем тестируемый метод для несуществующего заказа
	foundUID, err = repo.GetOrder("not-found")

	// Проверяем результаты
	assert.Error(t, err)
	assert.Equal(t, "", foundUID)
}

func TestPostgresRepository_GetAllOrders(t *testing.T) {
	// Создаем мок для базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Ошибка при создании мока БД: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close db", "error", err)
		}
	}()
	

	// Создаем репозиторий с моком БД
	repo := NewPostgresRepository(db)

	// Настраиваем ожидание запроса
	rows := sqlmock.NewRows([]string{"order_uid"}).
		AddRow("order-1").
		AddRow("order-2").
		AddRow("order-3")
	mock.ExpectQuery("SELECT order_uid FROM orders").
		WillReturnRows(rows)

	// Вызываем тестируемый метод
	orders, err := repo.GetAllOrders()

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Len(t, orders, 3)
	assert.Equal(t, []string{"order-1", "order-2", "order-3"}, orders)
	assert.NoError(t, mock.ExpectationsWereMet())

	// Тестируем случай с ошибкой
	expectedError := errors.New("database error")
	mock.ExpectQuery("SELECT order_uid FROM orders").
		WillReturnError(expectedError)

	// Вызываем тестируемый метод
	orders, err = repo.GetAllOrders()

	// Проверяем результаты
	assert.Error(t, err)
	assert.Nil(t, orders)
	assert.Equal(t, expectedError, err)
}
