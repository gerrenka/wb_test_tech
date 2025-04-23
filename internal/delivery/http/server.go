package httpserver

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"

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
	mux.HandleFunc("/order", s.getOrderHandler())

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.isRunning = true

	go func() {
		slog.Info("HTTP server started", "address", fmt.Sprintf("http://localhost%s", addr))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
		}
	}()

	return nil
}

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

func (s *OrderHTTPServer) renderOrderTemplate(w http.ResponseWriter, orderUID string) {
	tmplPath := filepath.Join("templates", "order.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		slog.Error("Failed to parse template", "error", err)
		http.Error(w, "Ошибка сервера при отображении шаблона", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, orderUID); err != nil {
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
	err := s.server.Shutdown(ctx)
	s.isRunning = false
	return err
}
