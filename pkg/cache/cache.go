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
}

// MemoryCache implements an in-memory LRU cache
type MemoryCache struct {
	mu    sync.RWMutex
	cache *lru.Cache[string, cacheValue]
}

type cacheValue struct {
	value interface{}
	exp   time.Time
}

// NewMemoryCache creates a new LRU memory cache
func NewMemoryCache(size int) *MemoryCache {
	lru, _ := lru.New[string, cacheValue](size)
	return &MemoryCache{
		cache: lru,
	}
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}

	if time.Now().After(val.exp) {
		c.cache.Remove(key)
		return nil, false
	}

	return val.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	exp := time.Now().Add(ttl)
	c.cache.Add(key, cacheValue{
		value: value,
		exp:   exp,
	})
}

func (c *MemoryCache) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.cache.Get(key)
	return ok
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
