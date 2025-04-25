package cache

import (
	"log/slog"
	"sync"
	"time"
)

// Cache представляет реализацию кэша в памяти
type Cache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
	ttl   time.Duration
}

type cacheItem struct {
	data      []byte
	createdAt time.Time
}

// NewCache создает новый экземпляр кэша с TTL
func NewCache(ttl time.Duration) *Cache {
	cache := &Cache{
		items: make(map[string]cacheItem),
		ttl:   ttl,
	}

	// Запускаем горутину для очистки устаревших элементов
	go cache.startCleanupTask()

	return cache
}

// Set добавляет ключ и данные в кэш
func (c *Cache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		data:      data,
		createdAt: time.Now(),
	}
	slog.Info("Cache updated", "key", key)
}

// Get получает данные из кэша
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	// Проверка TTL
	if c.ttl > 0 && time.Since(item.createdAt) > c.ttl {
		slog.Debug("Cache item expired", "key", key)
		return nil, false
	}

	return item.data, true
}

// Has проверяет наличие ключа в кэше
func (c *Cache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return false
	}

	// Проверка TTL
	if c.ttl > 0 && time.Since(item.createdAt) > c.ttl {
		slog.Debug("Cache item expired", "key", key)
		return false
	}

	slog.Debug("Cache check", "key", key, "found", true)
	return true
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

	slog.Info("Current cache content", "count", len(c.items))
	for key, item := range c.items {
		slog.Debug("Cache item", "key", key, "age", time.Since(item.createdAt))
	}
}

// Очистка устаревших элементов
func (c *Cache) startCleanupTask() {
	if c.ttl <= 0 {
		return // Если TTL не установлен, не запускаем очистку
	}

	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

func (c *Cache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Исправление: удалена неиспользуемая переменная now
	expiredKeys := 0

	for key, item := range c.items {
		if time.Since(item.createdAt) > c.ttl {
			delete(c.items, key)
			expiredKeys++
		}
	}

	if expiredKeys > 0 {
		slog.Info("Cache cleanup completed", "expired_keys", expiredKeys, "remaining", len(c.items))
	}
}
