package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
    "os/signal"
    "syscall"
	"time"

	"order-service/internal/config"
	"order-service/internal/infrastructure/cache"
	"order-service/internal/infrastructure/postgres"
    "order-service/internal/infrastructure/http"
    "order-service/internal/infrastructure/kafka"
	"order-service/internal/logger"
    "order-service/internal/usecase"
)

// Остальной код остается тем же

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
    defer func() {
        if err := db.Close(); err != nil {
            slog.Error("Failed to close database connection", "error", err)
        }
    }()
	repo := postgres.NewPostgresRepository(db)

	// Инициализация кеша
cacheRepo := cache.NewCache(cfg.CacheTTL)

// Загрузка заказов в кэш
orderUIDs, err := repo.GetAllOrders()
if err != nil {
    slog.Error("Failed to get orders for cache initialization", "error", err)
    slog.Warn("Continuing without initial cache population")
    orderUIDs = []string{} // Инициализируем пустым слайсом
} else {
    slog.Info("Initializing cache...", "order_count", len(orderUIDs))
    loadedCount := 0

    // Создаем канал для результатов загрузки
    type cacheLoadResult struct {
        orderUID string
        data     []byte
        err      error
    }

    resultChan := make(chan cacheLoadResult, len(orderUIDs))

    // Максимум 5 параллельных запросов
    sem := make(chan struct{}, 5)
    for _, orderUID := range orderUIDs {
        sem <- struct{}{} // блокируемся, если уже 5 горутин работают
        
        go func(uid string) {
            defer func() { <-sem }() // освобождаем слот в семафоре
            
            // Создаем отдельный контекст для каждого запроса
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()
            
            // Сначала попробуем получить кешированные данные
            cachedData, err := repo.GetCachedOrderData(uid)
            if err == nil && len(cachedData) > 0 {
                // Проверяем, что данные в кеше валидны
                var order struct{}
                if err := json.Unmarshal(cachedData, &order); err == nil {
                    resultChan <- cacheLoadResult{uid, cachedData, nil}
                    return
                }
            }
            
            // Если кешированных данных нет, получаем полный заказ
            order, err := repo.GetOrder(ctx, uid)
            if err != nil {
                resultChan <- cacheLoadResult{uid, nil, err}
                return
            }
            
            // Сериализуем заказ для кеша
            orderData, err := json.Marshal(order)
            if err != nil {
                resultChan <- cacheLoadResult{uid, nil, err}
                return
            }
            
            resultChan <- cacheLoadResult{uid, orderData, nil}
        }(orderUID)
    }

    // Ждем результаты и добавляем их в кеш
    for i := 0; i < len(orderUIDs); i++ {
        result := <-resultChan
        if result.err != nil {
            slog.Error("Failed to load order for cache", "error", result.err, "orderUID", result.orderUID)
            continue
        }
        
        // Добавляем в кеш
        cacheRepo.Set(result.orderUID, result.data)
        
        // Также сохраняем в БД для быстрого восстановления кеша
        if err := repo.CacheOrderData(result.orderUID, result.data); err != nil {
            slog.Error("Failed to save order cache to DB", "error", err, "orderUID", result.orderUID)
        }
        
        loadedCount++
    }

    // Ждем, пока все горутины завершатся
    for i := 0; i < cap(sem); i++ {
        sem <- struct{}{}
    }

    slog.Info("Cache initialized successfully", "loaded", loadedCount, "total", len(orderUIDs))
    // Инициализация приложения с нужными компонентами
app := usecase.NewApplication(
    cfg,
    repo,
    cacheRepo,
    kafka.NewOrderKafkaConsumer(
        cfg.KafkaBrokers,
        cfg.KafkaTopic,
        cfg.KafkaGroupID,
        repo,
        cacheRepo,
    ),
    http.NewOrderHTTPServer(
        cfg.ServerPort,
        repo,
        cacheRepo,
    ),
)

// Запуск приложения
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Запускаем все компоненты
if err := app.Start(ctx); err != nil {
    slog.Error("Failed to start application", "error", err)
    os.Exit(1)
}

// Обрабатываем сигналы для корректного завершения
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

// Ожидаем сигнала завершения
sig := <-sigChan
slog.Info("Received shutdown signal", "signal", sig)

// Корректно завершаем работу приложения
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer shutdownCancel()

if err := app.Shutdown(shutdownCtx); err != nil {
    slog.Error("Failed to shutdown application gracefully", "error", err)
    os.Exit(1)
}

slog.Info("Application shutdown completed")
}
}