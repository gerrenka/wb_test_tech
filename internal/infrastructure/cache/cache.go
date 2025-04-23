package cache

import (
	"log/slog"
	"sync"
)

// Cache представляет реализацию кэша в памяти
type Cache struct {
	mu    sync.RWMutex
	items map[string]struct{}
}

// NewCache создает новый экземпляр кэша
func NewCache() *Cache {
	return &Cache{
		items: make(map[string]struct{}),
	}
}

// Set добавляет ключ в кэш
func (c *Cache) Set(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = struct{}{}
	slog.Info("Cache updated", "key", key)
}

// Has проверяет наличие ключа в кэше
func (c *Cache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, found := c.items[key]
	slog.Debug("Cache check", "key", key, "found", found)
	return found
}

// Delete удаляет ключ из кэша
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	slog.Info("Removed from cache", "key", key)
}

// PrintContent выводит содержимое кэша в лог
func (c *Cache) PrintContent() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	slog.Info("Current cache content")
	for key := range c.items {
		slog.Debug("Cache item", "key", key)
	}
}
