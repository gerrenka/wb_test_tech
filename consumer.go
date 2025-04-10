package main

import (
	"context"
	"database/sql"
	//"encoding/json"
	// "fmt"
	// "html/template"
	"log/slog"
	// "net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	//"github.com/segmentio/kafka-go"
)

// Order represents the structure of messages from Kafka
type Order struct {
	OrderUID string `json:"order_uid"`
}

// Cache structure with mutex protection
type Cache struct {
	mu    sync.RWMutex
	items map[string]struct{}
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	return &Cache{
		items: make(map[string]struct{}),
	}
}

// Set adds an item to the cache
func (c *Cache) Set(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = struct{}{}
	slog.Info("Cache updated", "key", key)
}

// Has checks if an item exists in the cache
func (c *Cache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, found := c.items[key]
	slog.Debug("Cache check", "key", key, "found", found)
	return found
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	slog.Info("Removed from cache", "key", key)
}

// PrintContent dumps the cache content for debugging
func (c *Cache) PrintContent() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	slog.Info("Current cache content")
	for key := range c.items {
		slog.Debug("Cache item", "key", key)
	}
}

// Connect to PostgreSQL
func connectToDB(connStr string) (*sql.DB, error) {
	slog.Info("Connecting to database")
	return sql.Open("postgres", connStr)
}

func main() {
	// Initialize JSON logger
	InitLogger()
	
	slog.Info("Application starting")

	// Load configuration
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

	// Create application
	app, err := NewApplication(config)
	if err != nil {
		slog.Error("Failed to create application", "error", err)
		os.Exit(1)
	}

	// Print cache content for debugging
	app.cache.PrintContent()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start application
	if err := app.Start(ctx); err != nil {
		slog.Error("Failed to start application", "error", err)
		os.Exit(1)
	}

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-stop
	slog.Info("Shutdown signal received, gracefully shutting down...")

	// Create a deadline for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown application
	if err := app.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error during shutdown", "error", err)
	}

	slog.Info("Shutdown completed")
}