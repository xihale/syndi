package cache

import (
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2"
)

// Cache interface defines the cache operations
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Exists(key string) bool
	Delete(key string)
	Clear()
	Close() error
}

// MemoryCache implements an in-memory LRU cache.
// Values are stored decoded: no serialization happens on the hot path.
// Callers must treat retrieved values as read-only; the same instance is
// shared across Gets until eviction or expiry (same contract as promoteToMemory).
type MemoryCache struct {
	mu    sync.RWMutex
	cache *lru.Cache[string, cacheValue]
}

type cacheValue struct {
	value interface{}
	exp   time.Time
}

// cacheEntry wraps the value for gob encoding in the persistent layer.
type cacheEntry struct {
	Value interface{}
	Exp   time.Time
}

// NewMemoryCache creates a new LRU memory cache
func NewMemoryCache(size int) *MemoryCache {
	lruCache, _ := lru.New[string, cacheValue](size)
	return &MemoryCache{
		cache: lruCache,
	}
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	val, ok := c.cache.Get(key)

	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	// Check expiration
	if time.Now().After(val.exp) {
		c.mu.RUnlock()
		c.Delete(key)
		return nil, false
	}
	c.mu.RUnlock()

	return val.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Add(key, cacheValue{
		value: value,
		exp:   time.Now().Add(ttl),
	})
}

func (c *MemoryCache) Exists(key string) bool {
	c.mu.RLock()
	val, ok := c.cache.Get(key)
	c.mu.RUnlock()
	if !ok {
		return false
	}

	if time.Now().After(val.exp) {
		c.Delete(key)
		return false
	}

	return true
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Remove(key)
}

func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Purge()
}

// Len returns the number of items in the cache
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache.Len()
}

// Close implements the Cache interface - no-op for memory cache
func (c *MemoryCache) Close() error {
	return nil
}

// TryGet retrieves from cache or computes the value if missing
func (c *MemoryCache) TryGet(key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	val, err := fn()
	if err != nil {
		return nil, err
	}

	c.Set(key, val, ttl)
	return val, nil
}
