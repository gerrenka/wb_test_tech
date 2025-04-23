package main

import (
	"context"
	"log/slog"
	"order-service/internal/config"
	httpserver "order-service/internal/delivery/http"
	"order-service/internal/delivery/kafka"
	"order-service/internal/infrastructure/cache"
	"order-service/internal/infrastructure/postgres"
	"order-service/internal/logger"
	app "order-service/internal/usecase"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Инициализация логгера
	logger.InitLogger()
	slog.Info("Application starting")

	// Загрузка конфигурации
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("Configuration loaded",
		"db_host", cfg.DBHost,
		"db_port", cfg.DBPort,
		"kafka_topic", cfg.KafkaTopic,
		"server_port", cfg.ServerPort)

	// Инициализация репозитория БД
	db, err := postgres.ConnectToDB(cfg.GetDBConnString())
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	repo := postgres.NewPostgresRepository(db)

	// Инициализация кеша
	cacheRepo := cache.NewCache()

	// Загрузка заказов в кеш
	orders, err := repo.GetAllOrders()
	if err != nil {
		slog.Error("Failed to initialize cache", "error", err)
		os.Exit(1)
	}
	for _, orderUID := range orders {
		cacheRepo.Set(orderUID)
	}
	slog.Info("Cache initialized successfully", "count", len(orders))

	// Инициализация Kafka-консьюмера
	kafkaConsumer := kafka.NewOrderKafkaConsumer(
		cfg.KafkaBrokers,
		cfg.KafkaTopic,
		cfg.KafkaGroupID,
		repo,
		cacheRepo,
	)

	// Инициализация HTTP-сервера
	httpServer := httpserver.NewOrderHTTPServer(
		cfg.ServerPort,
		repo,
		cacheRepo,
	)

	// Инициализация приложения
	application := app.NewApplication(cfg, repo, cacheRepo, kafkaConsumer, httpServer)

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск компонентов
	if err := application.Start(ctx); err != nil {
		slog.Error("Failed to start application", "error", err)
		os.Exit(1)
	}

	// Обработка сигналов завершения
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	slog.Info("Shutdown signal received, gracefully shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error during shutdown", "error", err)
	}

	slog.Info("Shutdown completed")
}
