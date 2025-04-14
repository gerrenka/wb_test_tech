package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "html/template"
    "log/slog"
    "net/http"
    "time"

    "github.com/segmentio/kafka-go"
)

// CacheRepository представляет интерфейс для кеширования
type CacheRepository interface {
	Set(key string)
	Has(key string) bool
	Delete(key string)
	PrintContent()
}

// OrderRepository представляет интерфейс для работы с заказами в БД
type OrderRepository interface {
	SaveOrder(order Order) error
	GetOrder(orderUID string) (string, error)
	GetAllOrders() ([]string, error)
}

// PostgresRepository реализует OrderRepository для PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создает новый репозиторий PostgreSQL
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// SaveOrder сохраняет заказ в PostgreSQL
func (r *PostgresRepository) SaveOrder(order Order) error {
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

// GetOrder получает заказ из PostgreSQL по его идентификатору
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

// GetAllOrders получает все заказы из PostgreSQL
func (r *PostgresRepository) GetAllOrders() ([]string, error) {
	rows, err := r.db.Query("SELECT order_uid FROM orders")
	if err != nil {
		slog.Error("Failed to query orders", "error", err)
		return nil, err
	}
	defer rows.Close()

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

// KafkaConsumer представляет интерфейс для работы с Kafka
type KafkaConsumer interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// OrderKafkaConsumer реализует KafkaConsumer для обработки заказов
type OrderKafkaConsumer struct {
	brokers   []string
	topic     string
	groupID   string
	repo      OrderRepository
	cache     CacheRepository
	isRunning bool
	reader    *kafka.Reader
}

// NewOrderKafkaConsumer создает новый Kafka-консьюмер для заказов
func NewOrderKafkaConsumer(brokers []string, topic, groupID string, repo OrderRepository, cache CacheRepository) *OrderKafkaConsumer {
	return &OrderKafkaConsumer{
		brokers: brokers,
		topic:   topic,
		groupID: groupID,
		repo:    repo,
		cache:   cache,
	}
}

// Start запускает консьюмер Kafka
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

	// Запускаем обработку сообщений в горутине
	go c.processMessages(ctx)

	return nil
}

// processMessages обрабатывает сообщения из Kafka
func (c *OrderKafkaConsumer) processMessages(ctx context.Context) {
	for {
		// Проверяем отмену контекста
		select {
		case <-ctx.Done():
			slog.Info("Kafka consumer context canceled")
			return
		default:
			// Продолжаем обработку
		}

		// Используем тайм-аут для чтения сообщений
		msgCtx, msgCancel := context.WithTimeout(ctx, 1*time.Second)
		msg, err := c.reader.ReadMessage(msgCtx)
		msgCancel()

		if err != nil {
			if ctx.Err() != nil {
				// Контекст был отменен, выходим из цикла
				break
			}
			// Просто продолжаем при тайм-ауте или других ошибках
			continue
		}
		
		msgValue := string(msg.Value)
		slog.Info("Received message", "kafka_message", msgValue, "partition", msg.Partition, "offset", msg.Offset)

		var order Order
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
			continue
		}

		c.cache.Set(order.OrderUID)
		slog.Info("Message processed", "orderUID", order.OrderUID)
	}
}

// Shutdown останавливает консьюмер Kafka
func (c *OrderKafkaConsumer) Shutdown(ctx context.Context) error {
	if !c.isRunning || c.reader == nil {
		return nil
	}

	slog.Info("Kafka consumer shutting down")
	err := c.reader.Close()
	c.isRunning = false
	return err
}

// HTTPServer представляет интерфейс для HTTP-сервера
type HTTPServer interface {
	Start() error
	Shutdown(ctx context.Context) error
}

// OrderHTTPServer реализует HTTPServer для обработки заказов
type OrderHTTPServer struct {
	server   *http.Server
	repo     OrderRepository
	cache    CacheRepository
	port     int
	isRunning bool
}

// NewOrderHTTPServer создает новый HTTP-сервер для заказов
func NewOrderHTTPServer(port int, repo OrderRepository, cache CacheRepository) *OrderHTTPServer {
	return &OrderHTTPServer{
		port:  port,
		repo:  repo,
		cache: cache,
	}
}

// Start запускает HTTP-сервер
func (s *OrderHTTPServer) Start() error {
	if s.isRunning {
		return nil
	}

	addr := fmt.Sprintf(":%d", s.port)
	mux := http.NewServeMux()
	mux.HandleFunc("/order", s.getOrderHandler())

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.isRunning = true

	// Запускаем сервер в горутине
	go func() {
		slog.Info("HTTP server started", "address", fmt.Sprintf("http://localhost%s", addr))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
		}
	}()

	return nil
}

// getOrderHandler возвращает обработчик для получения информации о заказе
func (s *OrderHTTPServer) getOrderHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderUID := r.URL.Query().Get("id")
		if orderUID == "" {
			slog.Warn("Missing id parameter", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			http.Error(w, "Параметр 'id' обязателен", http.StatusBadRequest)
			return
		}

		slog.Info("Order lookup request", "orderUID", orderUID, "remote_addr", r.RemoteAddr)

		if s.cache.Has(orderUID) {
			slog.Info("Order found in cache", "orderUID", orderUID)
			s.renderOrderTemplate(w, orderUID)
			return
		}

		foundUID, err := s.repo.GetOrder(orderUID)
		if err == sql.ErrNoRows {
			slog.Info("Order not found", "orderUID", orderUID)
			http.Error(w, "Заказ не найден", http.StatusNotFound)
			return
		} else if err != nil {
			slog.Error("Database query error", "error", err, "orderUID", orderUID)
			http.Error(w, "Ошибка при запросе к БД", http.StatusInternalServerError)
			return
		}

		slog.Info("Order found in database", "orderUID", foundUID)
		s.renderOrderTemplate(w, foundUID)
	}
}

// renderOrderTemplate отображает шаблон с информацией о заказе
func (s *OrderHTTPServer) renderOrderTemplate(w http.ResponseWriter, orderUID string) {
	tmpl := template.Must(template.New("order").Parse(`
		<html>
		<head><title>Информация о заказе</title></head>
		<body>
			<h1>Заказ найден</h1>
			<p>Идентификатор заказа: {{.}}</p>
		</body>
		</html>
	`))
	tmpl.Execute(w, orderUID)
}

// Shutdown останавливает HTTP-сервер
func (s *OrderHTTPServer) Shutdown(ctx context.Context) error {
	if !s.isRunning || s.server == nil {
		return nil
	}

	slog.Info("HTTP server shutting down")
	err := s.server.Shutdown(ctx)
	s.isRunning = false
	return err
}

// Application представляет основное приложение
type Application struct {
	repo    OrderRepository
	cache   CacheRepository
	kafka   KafkaConsumer
	http    HTTPServer
	config  *Config
}

// NewApplication создает новое приложение
func NewApplication(config *Config) (*Application, error) {
	// Инициализация кеша
	cache := NewCache()

	// Подключение к БД
	db, err := connectToDB(config.GetDBConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Создание репозитория
	repo := NewPostgresRepository(db)

	// Инициализация кеша из БД
	orders, err := repo.GetAllOrders()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	for _, orderUID := range orders {
		cache.Set(orderUID)
	}
	slog.Info("Cache initialized successfully", "count", len(orders))

	// Создание Kafka-консьюмера
	kafka := NewOrderKafkaConsumer(
		config.KafkaBrokers,
		config.KafkaTopic,
		config.KafkaGroupID,
		repo,
		cache,
	)

	// Создание HTTP-сервера
	http := NewOrderHTTPServer(
		config.ServerPort,
		repo,
		cache,
	)

	return &Application{
		repo:    repo,
		cache:   cache,
		kafka:   kafka,
		http:    http,
		config:  config,
	}, nil
}

// Start запускает приложение
func (app *Application) Start(ctx context.Context) error {
	// Запуск Kafka-консьюмера
	if err := app.kafka.Start(ctx); err != nil {
		return fmt.Errorf("failed to start Kafka consumer: %w", err)
	}

	// Запуск HTTP-сервера
	if err := app.http.Start(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Shutdown останавливает приложение
func (app *Application) Shutdown(ctx context.Context) error {
	// Остановка HTTP-сервера
	if err := app.http.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	// Остановка Kafka-консьюмера
	if err := app.kafka.Shutdown(ctx); err != nil {
		slog.Error("Kafka consumer shutdown error", "error", err)
	}

	return nil
}