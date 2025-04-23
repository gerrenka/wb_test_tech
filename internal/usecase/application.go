package app

import (
	"context"
	"fmt"
	"log/slog"

	"order-service/internal/config"
	"order-service/pkg/interfaces"
)

type Application struct {
	repo   interfaces.OrderRepository
	cache  interfaces.CacheRepository
	kafka  interfaces.KafkaConsumer
	http   interfaces.HTTPServer
	config *config.Config
}

func NewApplication(config *config.Config, repo interfaces.OrderRepository, cache interfaces.CacheRepository, kafka interfaces.KafkaConsumer, http interfaces.HTTPServer) *Application {
	return &Application{
		repo:   repo,
		cache:  cache,
		kafka:  kafka,
		http:   http,
		config: config,
	}
}

func (app *Application) Start(ctx context.Context) error {
	if err := app.kafka.Start(ctx); err != nil {
		return fmt.Errorf("failed to start Kafka consumer: %w", err)
	}

	if err := app.http.Start(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

func (app *Application) Shutdown(ctx context.Context) error {
	if err := app.http.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	if err := app.kafka.Shutdown(ctx); err != nil {
		slog.Error("Kafka consumer shutdown error", "error", err)
	}

	return nil
}