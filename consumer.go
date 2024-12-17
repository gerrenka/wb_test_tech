package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"html/template"

	"github.com/segmentio/kafka-go"
	_ "github.com/lib/pq"
)

// Order represents the structure of messages from Kafka
type Order struct {
	OrderUID string `json:"order_uid"`
}

// Global cache
var cache = make(map[string]struct{})

// Connect to PostgreSQL
func connectToDB() (*sql.DB, error) {
	connStr := "host=localhost port=5434 user=my_user password=1 dbname=my_database sslmode=disable"
	return sql.Open("postgres", connStr)
}

// Save order to PostgreSQL


func saveOrderToDB(db *sql.DB, order Order) error {
	log.Printf("Запись в БД: OrderUID=%s", order.OrderUID)
	query := `
		INSERT INTO orders (order_uid)
		VALUES ($1)
		ON CONFLICT (order_uid) DO NOTHING;
	`
	_, err := db.Exec(query, order.OrderUID)
	if err != nil {
		log.Printf("Ошибка записи в БД: %v", err)
	}
	return err
}

// Update the cache
func updateCache(orderUID string) {
	cache[orderUID] = struct{}{}
	log.Printf("Кэш обновлён: Key=%s", orderUID)
}

// Check if order exists in cache
func checkCache(orderUID string) bool {
	_, found := cache[orderUID]
	log.Printf("Проверка в кэше: Key=%s, Найден=%v", orderUID, found)
	return found
}

// Initialize cache from database
func initializeCache(db *sql.DB) error {
	rows, err := db.Query("SELECT order_uid FROM orders")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return err
		}
		cache[orderUID] = struct{}{}
	}
	log.Println("Кэш успешно инициализирован из базы данных")
	return nil
}

// Print cache content for debugging
func printCache() {
	log.Println("Текущее содержимое кэша:")
	for key := range cache {
		log.Printf("Key: %s", key)
	}
}

// HTTP handler for fetching order details
func getOrderHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderUID := r.URL.Query().Get("id")
		if orderUID == "" {
			http.Error(w, "Параметр 'id' обязателен", http.StatusBadRequest)
			return
		}

		if checkCache(orderUID) {
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
			return
		}

		var foundUID string
		err := db.QueryRow("SELECT order_uid FROM orders WHERE order_uid = $1", orderUID).Scan(&foundUID)
		if err == sql.ErrNoRows {
			http.Error(w, "Заказ не найден", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Ошибка при запросе к БД", http.StatusInternalServerError)
			return
		}

		tmpl := template.Must(template.New("order").Parse(`
			<html>
			<head><title>Информация о заказе</title></head>
			<body>
				<h1>Заказ найден</h1>
				<p>Идентификатор заказа: {{.}}</p>
			</body>
			</html>
		`))
		tmpl.Execute(w, foundUID)
	}
}

func main() {
	db, err := connectToDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()

	if err := initializeCache(db); err != nil {
		log.Fatalf("Ошибка инициализации кэша: %v", err)
	}

	printCache()
	
	go func() {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "orders",
			GroupID: "order-consumer-group",
		})
		defer reader.Close()

		log.Println("Подписка на топик Kafka: orders")

		for {
			msg, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Ошибка чтения сообщения: %v", err)
				continue
			}
			log.Printf("Сырой JSON из Kafka: %s", string(msg.Value))


			log.Printf("Получено сообщение: %s", string(msg.Value))
			var order Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				log.Printf("Ошибка разбора сообщения: %v", err)
				continue
			}
			log.Printf("Сообщение разобрано успешно: %+v", order)


			if checkCache(order.OrderUID) {
				log.Printf("Заказ %s уже обработан, пропускаем", order.OrderUID)
				continue
			}

			if err := saveOrderToDB(db, order); err != nil {
				continue
			}

			updateCache(order.OrderUID)
			log.Printf("Сообщение обработано: %v", order)
		}
	}()

	http.HandleFunc("/order", getOrderHandler(db))
	log.Println("HTTP-сервер запущен на http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
