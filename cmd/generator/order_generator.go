package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	//"os"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Order represents the basic structure of an order message
type Order struct {
	OrderUID string `json:"order_uid"`
}

// Config holds the configuration for the order generator
type Config struct {
	KafkaBrokers  []string
	KafkaTopic    string
	Count         int
	Interval      time.Duration
	PrintOnly     bool
}

func main() {
	// Parse command line arguments
	var (
		brokers   = flag.String("brokers", "localhost:9093", "Kafka broker address")
		topic     = flag.String("topic", "orders", "Kafka topic to send orders to")
		count     = flag.Int("count", 10, "Number of orders to generate")
		interval  = flag.Int("interval", 1000, "Interval between orders in milliseconds")
		printOnly = flag.Bool("print-only", false, "Only print orders, don't send to Kafka")
	)
	flag.Parse()

	config := Config{
		KafkaBrokers:  []string{*brokers},
		KafkaTopic:    *topic,
		Count:         *count,
		Interval:      time.Duration(*interval) * time.Millisecond,
		PrintOnly:     *printOnly,
	}

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Create Kafka writer if needed
	var writer *kafka.Writer
	if !config.PrintOnly {
		writer = &kafka.Writer{
			Addr:     kafka.TCP(config.KafkaBrokers...),
			Topic:    config.KafkaTopic,
			Balancer: &kafka.LeastBytes{},
		}
		defer writer.Close()
		fmt.Printf("Connected to Kafka at %s\n", config.KafkaBrokers)
	}

	fmt.Printf("Generating %d orders with %v interval\n", config.Count, config.Interval)

	// Generate and send orders
	for i := 0; i < config.Count; i++ {
		// Generate a unique order ID
		orderUID := generateOrderUID()
		order := Order{OrderUID: orderUID}

		// Convert to JSON
		orderJSON, err := json.Marshal(order)
		if err != nil {
			log.Printf("Error marshaling order to JSON: %v", err)
			continue
		}

		if config.PrintOnly {
			// Just print the JSON
			fmt.Printf("Order %d: %s\n", i+1, string(orderJSON))
		} else {
			// Send to Kafka
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

		// Wait before sending the next order
		if i < config.Count-1 && config.Interval > 0 {
			time.Sleep(config.Interval)
		}
	}

	fmt.Println("Order generation completed")
}

// generateOrderUID creates a unique order identifier
func generateOrderUID() string {
	return uuid.New().String()
}