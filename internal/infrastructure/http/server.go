package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"order-service/internal/domain/models"
	"order-service/pkg/interfaces"
)

type OrderHTTPServer struct {
	server    *http.Server
	repo      interfaces.OrderRepository
	cache     interfaces.CacheRepository
	port      int
	isRunning bool
}

func NewOrderHTTPServer(port int, repo interfaces.OrderRepository, cache interfaces.CacheRepository) *OrderHTTPServer {
	return &OrderHTTPServer{
		port:  port,
		repo:  repo,
		cache: cache,
	}
}

func (s *OrderHTTPServer) Start() error {
	if s.isRunning {
		return nil
	}

	addr := fmt.Sprintf(":%d", s.port)
	mux := http.NewServeMux()

	// Регистрируем обработчики
	mux.HandleFunc("/order", s.getOrderHandler())
	mux.HandleFunc("/health", s.healthCheckHandler())

	// Middleware для логирования запросов
	wrappedHandler := s.loggingMiddleware(mux)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      wrappedHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.isRunning = true

	go func() {
		slog.Info("HTTP server started", "address", fmt.Sprintf("http://localhost%s", addr))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Улучшенное логирование при ошибке запуска
			slog.Error("HTTP server failed to start or encountered runtime error",
				"error", err,
				"address", addr,
				"port", s.port)

			// Можно добавить метрику или уведомление об отказе сервера
			// Например, увеличить счетчик ошибок или отправить уведомление
		}
	}()

	return nil
}

// Middleware для логирования
func (s *OrderHTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Оборачиваем ResponseWriter для получения статус-кода
		ww := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		// Вызываем следующий обработчик
		next.ServeHTTP(ww, r)

		// Логируем результат
		slog.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.statusCode,
			"duration", time.Since(start),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

// Обертка для ResponseWriter
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Обработчик для проверки работоспособности
func (s *OrderHTTPServer) healthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	}
}

// Обработчик для получения заказа
func (s *OrderHTTPServer) getOrderHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что метод GET
		if r.Method != http.MethodGet {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}

		orderUID := r.URL.Query().Get("id")
		if orderUID == "" {
			slog.Warn("Missing id parameter", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			http.Error(w, "Параметр 'id' обязателен", http.StatusBadRequest)
			return
		}

		slog.Info("Order lookup request", "orderUID", orderUID, "remote_addr", r.RemoteAddr)

		// Формат ответа (JSON или HTML)
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "html" // По умолчанию используем HTML
		}

		// Сначала проверяем кэш
		var order *models.Order
		cachedData, found := s.cache.Get(orderUID)

		if found {
			// Данные найдены в кэше
			var cachedOrder models.Order
			if err := json.Unmarshal(cachedData, &cachedOrder); err == nil {
				slog.Info("Order found in cache", "orderUID", orderUID)
				order = &cachedOrder
			} else {
				slog.Error("Failed to unmarshal cached order", "error", err)
				// Продолжаем и попробуем получить из БД
			}
		}

		// Если не нашли в кэше, ищем в БД
		if order == nil {
			var err error
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			order, err = s.repo.GetOrder(ctx, orderUID)
			if err == sql.ErrNoRows {
				slog.Info("Order not found", "orderUID", orderUID)
				http.Error(w, "Заказ не найден", http.StatusNotFound)
				return
			} else if err != nil {
				slog.Error("Database query error", "error", err, "orderUID", orderUID)
				http.Error(w, "Ошибка при запросе к БД", http.StatusInternalServerError)
				return
			}

			// Сохраняем заказ в кэш
			if order != nil {
				orderJSON, err := json.Marshal(order)
				if err == nil {
					s.cache.Set(orderUID, orderJSON)
				} else {
					slog.Error("Failed to marshal order for caching", "error", err)
				}
			}
		}

		// В зависимости от формата, возвращаем HTML или JSON
		if format == "json" {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(order); err != nil {
				slog.Error("Failed to encode order to JSON", "error", err)
				http.Error(w, "Ошибка сервера при отображении данных", http.StatusInternalServerError)
			}
			return
		}

		// По умолчанию - HTML
		s.renderOrderTemplate(w, order)
	}
}

func (s *OrderHTTPServer) renderOrderTemplate(w http.ResponseWriter, order *models.Order) {
	tmplPath := filepath.Join("templates", "order.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		slog.Error("Failed to parse template", "error", err)
		http.Error(w, "Ошибка сервера при отображении шаблона", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, order); err != nil {
		slog.Error("Failed to execute template", "error", err)
		http.Error(w, "Ошибка сервера при отображении данных", http.StatusInternalServerError)
		return
	}
}

func (s *OrderHTTPServer) Shutdown(ctx context.Context) error {
	if !s.isRunning || s.server == nil {
		return nil
	}

	slog.Info("HTTP server shutting down")

	// Создаем контекст с таймаутом для корректного завершения
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Сначала пытаемся корректно завершить работу
	err := s.server.Shutdown(shutdownCtx)

	// Проверяем результат завершения
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("HTTP server shutdown timed out, some connections may be forcibly closed",
				"error", err,
				"timeout", "30s")
		} else {
			slog.Error("HTTP server shutdown error",
				"error", err)
		}

		// В случае проблемы с корректным завершением можно попробовать принудительно закрыть
		closeErr := s.server.Close()
		if closeErr != nil {
			slog.Error("Failed to forcibly close HTTP server",
				"error", closeErr)
			// Возвращаем оригинальную ошибку, а не ошибку закрытия
		}
	} else {
		slog.Info("HTTP server shutdown completed successfully")
	}

	s.isRunning = false
	return err
}
