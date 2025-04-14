package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	_ "github.com/lib/pq"
)

type Order struct {
	OrderUID string `json:"order_uid"`
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]struct{}
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]struct{}),
	}
}

func (c *Cache) Set(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = struct{}{}
	slog.Info("Cache updated", "key", key)
}

func (c *Cache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, found := c.items[key]
	slog.Debug("Cache check", "key", key, "found", found)
	return found
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	slog.Info("Removed from cache", "key", key)
}

func (c *Cache) PrintContent() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	slog.Info("Current cache content")
	for key := range c.items {
		slog.Debug("Cache item", "key", key)
	}
}

func connectToDB(connStr string) (*sql.DB, error) {
	slog.Info("Connecting to database")
	return sql.Open("postgres", connStr)
}

func main() {
	InitLogger()
	slog.Info("Application starting")
	config, err := NewConfig()

	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("Configuration loaded", 
		"db_host", config.DBHost,
		"db_port", config.DBPort,
		"kafka_topic", config.KafkaTopic,
		"server_port", config.ServerPort)

	app, err := NewApplication(config)
	if err != nil {
		slog.Error("Failed to create application", "error", err)
		os.Exit(1)
	}

	app.cache.PrintContent()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Start(ctx); err != nil {
		slog.Error("Failed to start application", "error", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	slog.Info("Shutdown signal received, gracefully shutting down...")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error during shutdown", "error", err)
	}

	slog.Info("Shutdown completed")
}