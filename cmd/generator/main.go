package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
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
        orderUID := generateOrderUID()
        order := models.Order{OrderUID: orderUID}
    

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
					Key:   []byte(orderUID),
					Value: orderJSON,
				},
			)
			if err != nil {
				log.Printf("Error sending message to Kafka: %v", err)
			} else {
				fmt.Printf("Order %d sent to Kafka: %s\n", i+1, string(orderJSON))
			}
		}

		if i < config.Count-1 && config.Interval > 0 {
			time.Sleep(config.Interval)
		}
	}

	fmt.Println("Order generation completed")
}

func generateOrderUID() string {
	return uuid.New().String()
}
