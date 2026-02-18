package routeutils

import (
	"fmt"
	"time"

	rssubcache "github.com/xihale/rsshub-go/pkg/cache"
	"github.com/xihale/rsshub-go/pkg/models"
)

// CacheResult represents cached response with metadata
type CacheResult struct {
	Value     interface{}
	Hit       bool
	TTL       time.Duration
	ExpiresAt time.Time
}

// CacheFeed caches entire feed object
// Note: This is a simple wrapper - in practice, routes should use the cache middleware
// or call the cache instance directly through the context
func CacheFeed(cacheInstance rssubcache.Cache, key string, ttl time.Duration, fn func() (*models.Feed, error)) (*models.Feed, error) {
	if cacheInstance == nil {
		// Cache not available, execute function directly
		return fn()
	}

	// Try to get from cache
	if cached, ok := cacheInstance.Get(key); ok {
		if feed, ok := cached.(*models.Feed); ok {
			return feed, nil
		}
	}

	// Execute function to get feed
	feed, err := fn()
	if err != nil {
		return nil, err
	}

	// Cache the result
	cacheInstance.Set(key, feed, ttl)

	return feed, nil
}

// CacheItems caches just the items slice
func CacheItems(cacheInstance rssubcache.Cache, key string, ttl time.Duration, fn func() ([]models.Item, error)) ([]models.Item, error) {
	if cacheInstance == nil {
		// Cache not available, execute function directly
		return fn()
	}

	// Try to get from cache
	if cached, ok := cacheInstance.Get(key); ok {
		if items, ok := cached.([]models.Item); ok {
			return items, nil
		}
	}

	// Execute function to get items
	items, err := fn()
	if err != nil {
		return nil, err
	}

	// Cache the result
	cacheInstance.Set(key, items, ttl)

	return items, nil
}

// CacheJSON caches arbitrary JSON-marshalable value
func CacheJSON(cacheInstance rssubcache.Cache, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if cacheInstance == nil {
		// Cache not available, execute function directly
		return fn()
	}

	// Try to get from cache
	if cached, ok := cacheInstance.Get(key); ok {
		return cached, nil
	}

	// Execute function to get value
	value, err := fn()
	if err != nil {
		return nil, err
	}

	// Cache the result
	cacheInstance.Set(key, value, ttl)

	return value, nil
}

// TryGet wraps cache's TryGet method
func TryGet(cacheInstance rssubcache.Cache, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if cacheInstance == nil {
		// Cache not available, execute function directly
		return fn()
	}

	// Use cache's TryGet method if available (MemoryCache has it)
	type tryGetCache interface {
		TryGet(key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error)
	}

	if tg, ok := cacheInstance.(tryGetCache); ok {
		return tg.TryGet(key, ttl, fn)
	}

	// Fallback: implement try-get pattern manually
	if cached, ok := cacheInstance.Get(key); ok {
		return cached, nil
	}

	value, err := fn()
	if err != nil {
		return nil, err
	}

	cacheInstance.Set(key, value, ttl)
	return value, nil
}

// GetFeedFromCache retrieves a feed from cache without fallback function
func GetFeedFromCache(cacheInstance rssubcache.Cache, key string) (*models.Feed, bool) {
	if cacheInstance == nil {
		return nil, false
	}

	if cached, ok := cacheInstance.Get(key); ok {
		if feed, ok := cached.(*models.Feed); ok {
			return feed, true
		}
	}

	return nil, false
}

// GetItemsFromCache retrieves items from cache without fallback function
func GetItemsFromCache(cacheInstance rssubcache.Cache, key string) ([]models.Item, bool) {
	if cacheInstance == nil {
		return nil, false
	}

	if cached, ok := cacheInstance.Get(key); ok {
		if items, ok := cached.([]models.Item); ok {
			return items, true
		}
	}

	return nil, false
}

// SetFeedInCache stores a feed in cache
func SetFeedInCache(cacheInstance rssubcache.Cache, key string, feed *models.Feed, ttl time.Duration) error {
	if cacheInstance == nil {
		return fmt.Errorf("cache not available")
	}
	cacheInstance.Set(key, feed, ttl)
	return nil
}

// SetItemsInCache stores items in cache
func SetItemsInCache(cacheInstance rssubcache.Cache, key string, items []models.Item, ttl time.Duration) error {
	if cacheInstance == nil {
		return fmt.Errorf("cache not available")
	}
	cacheInstance.Set(key, items, ttl)
	return nil
}

// InvalidateCacheEntry removes a specific entry from cache
func InvalidateCacheEntry(cacheInstance rssubcache.Cache, key string) error {
	if cacheInstance == nil {
		return fmt.Errorf("cache not available")
	}
	cacheInstance.Delete(key)
	return nil
}

// CacheWithResult wraps CacheFeed and returns CacheResult with metadata
func CacheWithResult(cacheInstance rssubcache.Cache, key string, ttl time.Duration, fn func() (*models.Feed, error)) (*CacheResult, error) {
	start := time.Now()
	result := &CacheResult{
		TTL:       ttl,
		ExpiresAt: start.Add(ttl),
	}

	if cacheInstance == nil {
		// Cache not available
		result.Hit = false
		feed, err := fn()
		if err != nil {
			return nil, err
		}
		result.Value = feed
		return result, nil
	}

	// Try to get from cache
	if cached, ok := cacheInstance.Get(key); ok {
		if feed, ok := cached.(*models.Feed); ok {
			result.Hit = true
			result.Value = feed
			return result, nil
		}
	}

	// Execute function to get feed
	feed, err := fn()
	if err != nil {
		return nil, err
	}
	result.Hit = false
	result.Value = feed

	// Cache the result
	cacheInstance.Set(key, feed, ttl)

	return result, nil
}

// GenerateCacheKey generates a cache key from components
func GenerateCacheKey(parts ...string) string {
	key := ""
	for _, part := range parts {
		if part != "" {
			if key != "" {
				key += ":"
			}
			key += part
		}
	}
	return key
}

// CacheExists checks if a key exists in cache
func CacheExists(cacheInstance rssubcache.Cache, key string) bool {
	if cacheInstance == nil {
		return false
	}
	return cacheInstance.Exists(key)
}

// ClearCache clears all cache entries
func ClearCache(cacheInstance rssubcache.Cache) error {
	if cacheInstance == nil {
		return fmt.Errorf("cache not available")
	}
	cacheInstance.Clear()
	return nil
}
