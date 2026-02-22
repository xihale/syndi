package cache

import (
	"bytes"
	"encoding/gob"
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

// MemoryCache implements an in-memory LRU cache
type MemoryCache struct {
	mu    sync.RWMutex
	cache *lru.Cache[string, cacheValue]
}

type cacheValue struct {
	data []byte // gob-encoded data
	exp  time.Time
}

// cacheEntry wraps the value for gob encoding
type cacheEntry struct {
	Value interface{}
	Exp   time.Time
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

	// Check expiration
	if time.Now().After(val.exp) {
		return nil, false
	}

	// Decode gob-encoded data into cacheEntry wrapper
	var entry cacheEntry
	if err := gob.NewDecoder(bytes.NewReader(val.data)).Decode(&entry); err != nil {
		return nil, false
	}

	return entry.Value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Encode value using gob - wrap in struct to preserve type information
	entry := cacheEntry{
		Value: value,
		Exp:   time.Now().Add(ttl),
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(entry); err != nil {
		return
	}

	c.cache.Add(key, cacheValue{
		data: buf.Bytes(),
		exp:  entry.Exp,
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
