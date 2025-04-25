package postgres

import (
	"context"
	"database/sql"
	"log/slog"
	"order-service/internal/domain/models"
	"time"

	_ "github.com/lib/pq"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) SaveOrder(ctx context.Context, order models.Order) error {
	// Начинаем транзакцию
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return err
	}

	// Переменная для определения, нужно ли делать rollback
	committed := false

	// В случае ошибки откатываем транзакцию
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				slog.Error("Failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	// 1. Сохраняем основную информацию о заказе
	slog.Info("Saving order to database", "orderUID", order.OrderUID)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (
			order_uid, track_number, entry, locale, internal_signature, 
			customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) ON CONFLICT (order_uid) DO NOTHING;
	`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale,
		order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard,
	)

	if err != nil {
		slog.Error("Failed to insert order", "error", err, "orderUID", order.OrderUID)
		return err
	}

	// 2. Сохраняем информацию о доставке
	_, err = tx.ExecContext(ctx, `
		INSERT INTO delivery (
			order_uid, name, phone, zip, city, address, region, email
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) ON CONFLICT (order_uid) DO UPDATE SET
			name = $2, phone = $3, zip = $4, city = $5, 
			address = $6, region = $7, email = $8;
	`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone,
		order.Delivery.Zip, order.Delivery.City, order.Delivery.Address,
		order.Delivery.Region, order.Delivery.Email,
	)

	if err != nil {
		slog.Error("Failed to insert delivery", "error", err, "orderUID", order.OrderUID)
		return err
	}

	// 3. Сохраняем информацию об оплате
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payment (
			transaction_id, order_uid, request_id, currency, provider,
			amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) ON CONFLICT (transaction_id) DO UPDATE SET
			request_id = $3, currency = $4, provider = $5, amount = $6,
			payment_dt = $7, bank = $8, delivery_cost = $9, goods_total = $10, custom_fee = $11;
	`,
		order.Payment.Transaction, order.OrderUID, order.Payment.RequestID,
		order.Payment.Currency, order.Payment.Provider, order.Payment.Amount,
		order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost,
		order.Payment.GoodsTotal, order.Payment.CustomFee,
	)

	if err != nil {
		slog.Error("Failed to insert payment", "error", err, "orderUID", order.OrderUID)
		return err
	}

	// 4. Сохраняем товары
	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO items (
				chrt_id, order_uid, track_number, price, rid, name,
				sale, size, total_price, nm_id, brand, status
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
			) ON CONFLICT (chrt_id) DO UPDATE SET
				track_number = $3, price = $4, rid = $5, name = $6, sale = $7,
				size = $8, total_price = $9, nm_id = $10, brand = $11, status = $12;
		`,
			item.ChrtID, order.OrderUID, item.TrackNumber, item.Price,
			item.Rid, item.Name, item.Sale, item.Size, item.TotalPrice,
			item.NmID, item.Brand, item.Status,
		)

		if err != nil {
			slog.Error("Failed to insert item", "error", err, "orderUID", order.OrderUID, "chrtID", item.ChrtID)
			return err
		}
	}

	// Фиксируем транзакцию
	if err = tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction", "error", err, "orderUID", order.OrderUID)
		return err
	}
	committed = true
	slog.Info("Order successfully saved", "orderUID", order.OrderUID)
	return nil
}

func (r *PostgresRepository) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	// Запрос для получения основной информации о заказе
	orderRow := r.db.QueryRowContext(ctx, `
		SELECT 
			order_uid, track_number, entry, locale, internal_signature, 
			customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders 
		WHERE order_uid = $1
	`, orderUID)

	var order models.Order
	var dateCreated string
	err := orderRow.Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
		&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
		&order.Shardkey, &order.SmID, &dateCreated, &order.OofShard,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			slog.Info("Order not found in database", "orderUID", orderUID)
		} else {
			slog.Error("Error querying order", "error", err, "orderUID", orderUID)
		}
		return nil, err
	}

	// Парсинг даты создания
	order.DateCreated, err = time.Parse(time.RFC3339, dateCreated)
	if err != nil {
		slog.Error("Error parsing date", "error", err, "date", dateCreated)
		return nil, err
	}

	// Запрос для получения информации о доставке
	deliveryRow := r.db.QueryRowContext(ctx, `
		SELECT 
			name, phone, zip, city, address, region, email
		FROM delivery 
		WHERE order_uid = $1
	`, orderUID)

	err = deliveryRow.Scan(
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
		&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region,
		&order.Delivery.Email,
	)

	if err != nil {
		slog.Error("Error querying delivery", "error", err, "orderUID", orderUID)
		return nil, err
	}

	// Запрос для получения информации об оплате
	paymentRow := r.db.QueryRowContext(ctx, `
		SELECT 
			transaction_id, request_id, currency, provider, amount,
			payment_dt, bank, delivery_cost, goods_total, custom_fee
		FROM payment 
		WHERE order_uid = $1
	`, orderUID)

	err = paymentRow.Scan(
		&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
		&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
		&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal,
		&order.Payment.CustomFee,
	)

	if err != nil {
		slog.Error("Error querying payment", "error", err, "orderUID", orderUID)
		return nil, err
	}

	// Запрос для получения товаров
	itemsRows, err := r.db.QueryContext(ctx, `
		SELECT 
			chrt_id, track_number, price, rid, name, sale,
			size, total_price, nm_id, brand, status
		FROM items 
		WHERE order_uid = $1
	`, orderUID)

	if err != nil {
		slog.Error("Error querying items", "error", err, "orderUID", orderUID)
		return nil, err
	}
	defer func() {
		if err := itemsRows.Close(); err != nil {
			slog.Error("failed to close items rows", "error", err)
		}
	}()

	order.Items = make([]models.Item, 0)
	for itemsRows.Next() {
		var item models.Item
		if err := itemsRows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid,
			&item.Name, &item.Sale, &item.Size, &item.TotalPrice,
			&item.NmID, &item.Brand, &item.Status,
		); err != nil {
			slog.Error("Error scanning item", "error", err)
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	if err = itemsRows.Err(); err != nil {
		slog.Error("Error iterating items rows", "error", err)
		return nil, err
	}

	slog.Info("Order retrieved successfully", "orderUID", orderUID)
	return &order, nil
}

func (r *PostgresRepository) GetAllOrders() ([]string, error) {
	rows, err := r.db.Query("SELECT order_uid FROM orders")
	if err != nil {
		slog.Error("Failed to query orders", "error", err)
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("failed to close rows", "error", err)
		}
	}()

	var orders []string
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			slog.Error("Failed to scan order_uid", "error", err)
			return nil, err
		}
		orders = append(orders, orderUID)
	}

	if err = rows.Err(); err != nil {
		slog.Error("Error iterating order rows", "error", err)
		return nil, err
	}

	return orders, nil
}

// Новые методы для кеширования полных данных заказа
func (r *PostgresRepository) CacheOrderData(orderUID string, orderData []byte) error {
	_, err := r.db.Exec(`
		INSERT INTO order_cache (order_uid, data, created_at) 
		VALUES ($1, $2, NOW())
		ON CONFLICT (order_uid) DO UPDATE SET 
		data = $2, created_at = NOW()
	`, orderUID, orderData)

	if err != nil {
		slog.Error("Failed to cache order data", "error", err, "orderUID", orderUID)
	}

	return err
}

func (r *PostgresRepository) GetCachedOrderData(orderUID string) ([]byte, error) {
	var data []byte
	err := r.db.QueryRow(`
		SELECT data FROM order_cache WHERE order_uid = $1
	`, orderUID).Scan(&data)

	if err != nil {
		if err != sql.ErrNoRows {
			slog.Error("Failed to get cached order data", "error", err, "orderUID", orderUID)
		}
		return nil, err
	}

	return data, nil
}

// ConnectToDB устанавливает подключение к базе данных
func ConnectToDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Устанавливаем параметры пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
