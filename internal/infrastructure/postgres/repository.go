package postgres

import (
	"database/sql"
	"log/slog"
	"order-service/internal/domain/models"

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

func (r *PostgresRepository) SaveOrder(order models.Order) error {
	slog.Info("Saving to database", "orderUID", order.OrderUID)
	query := `
		INSERT INTO orders (order_uid)
		VALUES ($1)
		ON CONFLICT (order_uid) DO NOTHING;
	`
	_, err := r.db.Exec(query, order.OrderUID)
	if err != nil {
		slog.Error("Database write error", "error", err, "orderUID", order.OrderUID)
	}
	return err
}

func (r *PostgresRepository) GetOrder(orderUID string) (string, error) {
	var foundUID string
	err := r.db.QueryRow("SELECT order_uid FROM orders WHERE order_uid = $1", orderUID).Scan(&foundUID)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Info("Order not found in database", "orderUID", orderUID)
		} else {
			slog.Error("Database query error", "error", err, "orderUID", orderUID)
		}
		return "", err
	}
	return foundUID, nil
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
	return orders, nil
}

// ConnectToDB устанавливает подключение к базе данных
func ConnectToDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
