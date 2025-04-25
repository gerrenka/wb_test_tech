package tests

import (
    "context"
    "database/sql"
    "errors"
    "testing"
    "log/slog"
    
    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/assert"
    
    "order-service/internal/domain/models"
    "order-service/internal/infrastructure/postgres"
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
    repo := postgres.NewPostgresRepository(db)
    
    // Тестовый заказ
    order := models.Order{OrderUID: "test-order-123"}
    ctx := context.Background()
    
    // Настраиваем ожидание запроса
    // Обратите внимание, что мы должны настроить ожидание в соответствии с реальным SQL-запросом из репозитория
    mock.ExpectExec("INSERT INTO orders").
        WithArgs(order.OrderUID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 
                sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 
                sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))
    
    // Для метода SaveOrder нужно также настроить ожидания для вставки в delivery, payment и items
    // Имитируем запросы для вставки данных доставки
    mock.ExpectExec("INSERT INTO delivery").
        WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
               sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))
    
    // Имитируем запросы для вставки данных оплаты
    mock.ExpectExec("INSERT INTO payment").
        WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
               sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
               sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))
    
    // Для items также нужно настроить ожидания, если есть товары в заказе
    
    // Вызываем тестируемый метод
    err = repo.SaveOrder(ctx, order)
    
    // Проверяем результаты
    assert.NoError(t, err)
    assert.NoError(t, mock.ExpectationsWereMet())
    
    // Тестируем случай с ошибкой
    expectedError := errors.New("database error")
    mock.ExpectExec("INSERT INTO orders").
        WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
               sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
               sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnError(expectedError)
    
    // Вызываем тестируемый метод с ошибкой
    err = repo.SaveOrder(ctx, models.Order{OrderUID: "error-order"})
    
    // Проверяем результаты
    assert.Error(t, err)
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
    repo := postgres.NewPostgresRepository(db)
    
    // Тестовый ID заказа
    orderUID := "test-order-456"
    ctx := context.Background()
    
    // Настраиваем ожидания для всех запросов, которые делает метод GetOrder
    // Основные данные заказа
    orderRows := sqlmock.NewRows([]string{
        "order_uid", "track_number", "entry", "locale", "internal_signature", 
        "customer_id", "delivery_service", "shardkey", "sm_id", "date_created", "oof_shard"
    }).AddRow(
        orderUID, "track123", "WBIL", "en", "", 
        "customer123", "meest", "9", 99, "2021-11-26T06:22:19Z", "1"
    )
    
    mock.ExpectQuery("SELECT .* FROM orders WHERE order_uid = \\$1").
        WithArgs(orderUID).
        WillReturnRows(orderRows)
    
    // Данные доставки
    deliveryRows := sqlmock.NewRows([]string{
        "name", "phone", "zip", "city", "address", "region", "email"
    }).AddRow(
        "Test User", "+7123456789", "123456", "Moscow", "123 Test St", "Test Region", "test@example.com"
    )
    
    mock.ExpectQuery("SELECT .* FROM delivery WHERE order_uid = \\$1").
        WithArgs(orderUID).
        WillReturnRows(deliveryRows)
    
    // Данные оплаты
    paymentRows := sqlmock.NewRows([]string{
        "transaction_id", "request_id", "currency", "provider", "amount",
        "payment_dt", "bank", "delivery_cost", "goods_total", "custom_fee"
    }).AddRow(
        "trans123", "", "USD", "wbpay", 1500,
        1637907727, "sber", 200, 1300, 0
    )
    
    mock.ExpectQuery("SELECT .* FROM payment WHERE order_uid = \\$1").
        WithArgs(orderUID).
        WillReturnRows(paymentRows)
    
    // Данные товаров
    itemsRows := sqlmock.NewRows([]string{
        "chrt_id", "track_number", "price", "rid", "name", "sale",
        "size", "total_price", "nm_id", "brand", "status"
    }).AddRow(
        9934930, "track123", 300, "rid123", "Test Item", 10,
        "M", 270, 2389212, "Test Brand", 202
    )
    
    mock.ExpectQuery("SELECT .* FROM items WHERE order_uid = \\$1").
        WithArgs(orderUID).
        WillReturnRows(itemsRows)
    
    // Вызываем тестируемый метод
    order, err := repo.GetOrder(ctx, orderUID)
    
    // Проверяем результаты
    assert.NoError(t, err)
    assert.NotNil(t, order)
    assert.Equal(t, orderUID, order.OrderUID)
    assert.NoError(t, mock.ExpectationsWereMet())
    
    // Тестируем случай, когда заказ не найден
    mock.ExpectQuery("SELECT .* FROM orders WHERE order_uid = \\$1").
        WithArgs("not-found").
        WillReturnError(sql.ErrNoRows)
    
    // Вызываем тестируемый метод для несуществующего заказа
    order, err = repo.GetOrder(ctx, "not-found")
    
    // Проверяем результаты
    assert.Error(t, err)
    assert.Nil(t, order)
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
    repo := postgres.NewPostgresRepository(db)
    
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