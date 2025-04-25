package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	
	"order-service/internal/domain/models"
)

func main() {
	var (
		brokers   = flag.String("brokers", "localhost:9093", "Kafka broker address")
		topic     = flag.String("topic", "orders", "Kafka topic to send orders to")
		count     = flag.Int("count", 10, "Number of orders to generate")
		interval  = flag.Int("interval", 1000, "Interval between orders in milliseconds")
		printOnly = flag.Bool("print-only", false, "Only print orders, don't send to Kafka")
	)
	flag.Parse()

	// Использовать переменные окружения, если они указаны
	if brokersEnv := os.Getenv("BROKERS"); brokersEnv != "" {
		*brokers = brokersEnv
	}
	if topicEnv := os.Getenv("TOPIC"); topicEnv != "" {
		*topic = topicEnv
	}
	if countEnv := os.Getenv("COUNT"); countEnv != "" {
		var err error
		*count, err = fmt.Sscanf(countEnv, "%d", count)
		if err != nil {
			slog.Error("Failed to parse COUNT env var", "error", err)
		}
	}
	if intervalEnv := os.Getenv("INTERVAL"); intervalEnv != "" {
		var err error
		*interval, err = fmt.Sscanf(intervalEnv, "%d", interval)
		if err != nil {
			slog.Error("Failed to parse INTERVAL env var", "error", err)
		}
	}

	config := models.GeneratorConfig{
		KafkaBrokers: []string{*brokers},
		KafkaTopic:   *topic,
		Count:        *count,
		Interval:     time.Duration(*interval) * time.Millisecond,
		PrintOnly:    *printOnly,
	}

	var writer *kafka.Writer
	if !config.PrintOnly {
		writer = &kafka.Writer{
			Addr:     kafka.TCP(config.KafkaBrokers...),
			Topic:    config.KafkaTopic,
			Balancer: &kafka.LeastBytes{},
		}
		defer func() {
			if err := writer.Close(); err != nil {
				slog.Error("failed to close writer", "error", err)
			}
		}()
		
		fmt.Printf("Connected to Kafka at %s\n", config.KafkaBrokers)
	}

	fmt.Printf("Generating %d orders with %v interval\n", config.Count, config.Interval)

	for i := 0; i < config.Count; i++ {
		order := generateRandomOrder()

		orderJSON, err := json.Marshal(order)
		if err != nil {
			log.Printf("Error marshaling order to JSON: %v", err)
			continue
		}

		if config.PrintOnly {
			fmt.Printf("Order %d: %s\n", i+1, string(orderJSON))
		} else {
			err = writer.WriteMessages(context.Background(),
				kafka.Message{
					Key:   []byte(order.OrderUID),
					Value: orderJSON,
				},
			)
			if err != nil {
				log.Printf("Error sending message to Kafka: %v", err)
			} else {
				fmt.Printf("Order %d sent to Kafka: %s\n", i+1, order.OrderUID)
			}
		}

		if i < config.Count-1 && config.Interval > 0 {
			time.Sleep(config.Interval)
		}
	}

	fmt.Println("Order generation completed")
}

// Генерация случайного заказа
func generateRandomOrder() models.Order {
	orderUID := uuid.New().String()
	trackNumber := fmt.Sprintf("TRACK%d", rand.Intn(1000000))
	now := time.Now()
	
	return models.Order{
		OrderUID:    orderUID,
		TrackNumber: trackNumber,
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    fmt.Sprintf("Customer %d", rand.Intn(1000)),
			Phone:   fmt.Sprintf("+7%09d", rand.Intn(1000000000)),
			Zip:     fmt.Sprintf("%06d", rand.Intn(1000000)),
			City:    randomCity(),
			Address: fmt.Sprintf("ул. %s, д. %d", randomStreet(), rand.Intn(100)+1),
			Region:  randomRegion(),
			Email:   fmt.Sprintf("user%d@example.com", rand.Intn(10000)),
		},
		Payment: models.Payment{
			Transaction:  orderUID,
			RequestID:    "",
			Currency:     randomCurrency(),
			Provider:     "wbpay",
			Amount:       rand.Intn(10000) + 1000,
			PaymentDt:    now.Unix(),
			Bank:         randomBank(),
			DeliveryCost: float64(rand.Intn(500) + 100),
			GoodsTotal:   float64(rand.Intn(9000) + 500),
			CustomFee:    0,
		},
		Items:             generateRandomItems(trackNumber, rand.Intn(5)+1),
		Locale:            randomLocale(),
		InternalSignature: "",
		CustomerID:        fmt.Sprintf("customer_%d", rand.Intn(10000)),
		DeliveryService:   randomDeliveryService(),
		Shardkey:          fmt.Sprintf("%d", rand.Intn(10)),
		SmID:              rand.Intn(100),
		DateCreated:       now,
		OofShard:          fmt.Sprintf("%d", rand.Intn(5)),
	}
}

// Генерация случайных товаров
func generateRandomItems(trackNumber string, count int) []models.Item {
	items := make([]models.Item, count)
	
	for i := 0; i < count; i++ {
		price := float64(rand.Intn(5000) + 100)
		sale := rand.Intn(50)
		totalPrice := price * (1 - float64(sale)/100)
		
		items[i] = models.Item{
			ChrtID:      int64(rand.Intn(10000000) + 1000000),
			TrackNumber: trackNumber,
			Price:       price,
			Rid:         uuid.New().String(),
			Name:        randomProductName(),
			Sale:        sale,
			Size:        fmt.Sprintf("%d", rand.Intn(50)),
			TotalPrice:  totalPrice,
			NmID:        int64(rand.Intn(10000000) + 1000000),
			Brand:       randomBrand(),
			Status:      rand.Intn(500) + 100,
		}
	}
	
	return items
}

// Вспомогательные функции для генерации случайных данных
func randomCity() string {
	cities := []string{
		"Москва", "Санкт-Петербург", "Новосибирск", "Екатеринбург", 
		"Казань", "Нижний Новгород", "Самара", "Омск", 
		"Ростов-на-Дону", "Уфа",
	}
	return cities[rand.Intn(len(cities))]
}

func randomStreet() string {
	streets := []string{
		"Ленина", "Мира", "Гагарина", "Пушкина", "Советская", 
		"Молодежная", "Центральная", "Школьная", "Лесная", "Садовая",
	}
	return streets[rand.Intn(len(streets))]
}

func randomRegion() string {
	regions := []string{
		"Московская область", "Ленинградская область", "Свердловская область", 
		"Новосибирская область", "Республика Татарстан", "Краснодарский край",
	}
	return regions[rand.Intn(len(regions))]
}

func randomCurrency() string {
	currencies := []string{"RUB", "USD", "EUR"}
	return currencies[rand.Intn(len(currencies))]
}

func randomBank() string {
	banks := []string{"sber", "alpha", "tinkoff", "vtb", "raiffeisen"}
	return banks[rand.Intn(len(banks))]
}

func randomLocale() string {
	locales := []string{"ru", "en", "de"}
	return locales[rand.Intn(len(locales))]
}

func randomDeliveryService() string {
	services := []string{"meest", "dhl", "fedex", "usps", "ups"}
	return services[rand.Intn(len(services))]
}

func randomProductName() string {
	products := []string{
		"Футболка", "Джинсы", "Куртка", "Платье", "Рубашка", 
		"Брюки", "Юбка", "Свитер", "Шапка", "Перчатки",
		"Наушники", "Смартфон", "Ноутбук", "Клавиатура", "Мышь", 
		"Монитор", "Планшет", "Колонки", "Часы", "Зарядка",
		"Кроссовки", "Ботинки", "Туфли", "Сандалии", "Кеды", 
		"Тапочки", "Шлепанцы", "Сапоги",
	}
	return products[rand.Intn(len(products))]
}

func randomBrand() string {
	brands := []string{
		"Nike", "Adidas", "Puma", "Reebok", "Samsung", "Apple", 
		"Xiaomi", "Huawei", "LG", "Sony", "Zara", "H&M", 
		"Pull&Bear", "Bershka", "ASOS", "Lacoste", 
		"Tommy Hilfiger", "Calvin Klein", "Lenovo", "HP", 
		"Dell", "Asus", "Logitech", "JBL", "Sennheiser", 
		"Bosch", "Philips",
	}
	return brands[rand.Intn(len(brands))]
}

func init() {
	// Инициализация генератора случайных чисел
	rand.Seed(time.Now().UnixNano())
}