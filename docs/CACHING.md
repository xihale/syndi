# Caching in rsshub-go

rsshub-go provides two approaches for caching RSS feed responses:

## Table of Contents

- [Overview](#overview)
- [Middleware-Level Caching](#middleware-level-caching)
- [Handler-Level Caching (Recommended)](#handler-level-caching-recommended)
- [Comparison](#comparison)
- [Migration Guide](#migration-guide)
- [Configuration](#configuration)
- [Cache Invalidation](#cache-invalidation)

---

## Overview

Both caching approaches provide:

- ✅ **Response caching** with configurable TTL
- ✅ **ETag support** for 304 Not Modified responses
- ✅ **X-Cache headers** (HIT/MISS) for debugging
- ✅ **Smart bypass logic** for static assets
- ✅ **Format support** (RSS, Atom, JSON)

---

## Middleware-Level Caching

**Location**: `internal/middleware/cache.go`

The middleware-level caching applies to all routes automatically.

### Pros

- ✅ Automatic - no code changes needed per route
- ✅ Centralized configuration
- ✅ Request deduplication (prevents cache stampede)

### Cons

- ❌ Response body interception issues with Gin
- ❌ Limited per-route customization
- ❌ Cache status headers may not be visible in all cases
- ❌ All-or-nothing approach

### Usage

Already configured in `cmd/server.go`:

```go
engine.Use(
    middleware.Recovery(),
    middleware.Logger(),
    middleware.Header(cfg.CacheTTL),
    middleware.Cache(cacheInstance, cfg.CacheTTL),
    middleware.Parameter(),
)
```

### Known Limitations

Due to Gin's internal handling of response writers, the middleware approach has limitations:
- Cache status headers (X-Cache) may not propagate correctly
- Response body buffering can cause issues
- Less control over individual route behavior

---

## Handler-Level Caching (Recommended)

**Location**: `internal/cache/handler.go`

The handler-level caching wraps individual route handlers with caching logic.

### Pros

- ✅ **No writer wrapping issues**
- ✅ Full per-route customization
- ✅ Cache headers always visible
- ✅ Flexible TTL and key generation per route
- ✅ Conditional caching based on response content

### Cons

- ❌ Requires explicit opt-in per route
- ❌ More verbose configuration

### Basic Usage

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/rsshub/go/internal/cache"
    "github.com/rsshub/go/pkg/models"
)

// Simple: use defaults
engine.GET("/feed", cache.NewCachedHandler(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
    return &models.Feed{Title: "My Feed"}, nil
}))

// With custom TTL (5 minutes)
engine.GET("/news", cache.NewCachedHandlerWithTTL(cacheInstance, handler, 5*time.Minute))

// With full options
opts := &cache.CachedHandlerOptions{
    TTL:         30 * time.Minute,
    ETagEnabled:  true,
    KeyGenerator: func(c *gin.Context) string {
        return "custom:" + c.Request.URL.Path
    },
    ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
        // Only cache if feed has items
        return feed != nil && len(feed.Items) > 0
    },
}

engine.GET("/custom", cache.Cached(cacheInstance, handler, opts))
```

### Real-World Example

```go
// In your route registration
func RegisterRoutes(engine *gin.Engine, cacheInstance cache.Cache, registry *RouteRegistry) {
    // Status endpoint - no caching
    engine.GET("/status", handleStatus)

    // News feed - short cache (5 min)
    engine.GET("/news", cache.NewCachedHandlerWithTTL(cacheInstance, handleNews, 5*time.Minute))

    // Documentation - long cache (1 hour)
    engine.GET("/docs/:page", cache.NewCachedHandlerWithTTL(cacheInstance, handleDocs, 1*time.Hour))

    // User feed - custom key includes user ID
    opts := &cache.CachedHandlerOptions{
        TTL: 30 * time.Minute,
        KeyGenerator: func(c *gin.Context) string {
            userID := c.Param("id")
            return "user:" + userID + ":" + c.Request.URL.Path
        },
        ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
            // Don't cache empty feeds
            return feed != nil && len(feed.Items) > 0
        },
    }
    engine.GET("/user/:id/feed", cache.Cached(cacheInstance, handleUserFeed, opts))
}
```

---

## Comparison

| Feature | Middleware | Handler-Level |
|---------|-----------|---------------|
| **Ease of use** | ⭐⭐⭐⭐⭐ (automatic) | ⭐⭐⭐ (manual) |
| **Per-route control** | ⭐⭐ (limited) | ⭐⭐⭐⭐⭐ (full) |
| **Header visibility** | ⭐⭐ (issues) | ⭐⭐⭐⭐⭐ (always works) |
| **Flexibility** | ⭐⭐⭐ (moderate) | ⭐⭐⭐⭐⭐ (high) |
| **Request deduplication** | ⭐⭐⭐⭐⭐ (built-in) | ⭐⭐ (manual) |
| **Production ready** | ⭐⭐⭐ (with caveats) | ⭐⭐⭐⭐⭐ (fully) |

---

## Migration Guide

### From Middleware to Handler-Level

**Before** (middleware approach):

```go
// In cmd/server.go
engine.Use(middleware.Cache(cacheInstance, cfg.CacheTTL))

engine.GET("/feed", myHandler)
```

**After** (handler-level approach):

```go
// Remove middleware caching
engine.Use(
    middleware.Recovery(),
    middleware.Logger(),
    middleware.Header(cfg.CacheTTL),
    // middleware.Cache() - remove this
    middleware.Parameter(),
)

// Wrap individual handlers
engine.GET("/feed", cache.NewCachedHandler(cacheInstance, myHandler))
```

### Gradual Migration

You can use both approaches simultaneously:

```go
// Keep middleware for some routes
engine.Use(middleware.Cache(cacheInstance, cfg.CacheTTL))

// Override with handler-level for specific routes
engine.GET("/important", cache.NewCachedHandler(cacheInstance, importantHandler, 1*time.Hour))
```

---

## Configuration

### Environment Variables

```bash
# Enable/disable caching globally
ENABLE_CACHE=true

# Cache TTL (default: 15 minutes)
CACHE_TTL=900

# Cache access key (if using access control)
ACCESS_KEY=your_secret_key
```

### Per-Route Configuration

```go
// Fast-changing data
opts1 := &cache.CachedHandlerOptions{
    TTL: 5 * time.Minute,
}

// Slow-changing data
opts2 := &cache.CachedHandlerOptions{
    TTL: 1 * time.Hour,
}

// Never cache
opts3 := &cache.CachedHandlerOptions{
    ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
        return false // Never cache
    },
}
```

---

## Cache Invalidation

### Manual Invalidation

```go
import "github.com/rsshub/go/internal/cache"

// Invalidate specific route
func invalidateFeed(c *gin.Context, cacheInstance cache.Cache) {
    cache.InvalidateCache(cacheInstance, c)
}

// Usage in handler
engine.POST("/feed/:id/refresh", func(c *gin.Context) {
    // Update feed data
    updateFeed(c.Param("id"))

    // Invalidate cache
    cache.InvalidateCache(cacheInstance, c)

    c.JSON(http.StatusOK, gin.H{"status": "refreshed"})
})
```

### Automatic Invalidation (TTL)

All cached responses automatically expire after their TTL.

---

## Advanced Features

### Custom Cache Key Generation

```go
opts := &cache.CachedHandlerOptions{
    KeyGenerator: func(c *gin.Context) string {
        // Include user-specific data
        userID := c.GetHeader("X-User-ID")

        // Include format preference
        format := c.Query("format")

        // Include query parameters
        limit := c.Query("limit")

        key := "feed:user:" + userID
        if format != "" {
            key += ":" + format
        }
        if limit != "" {
            key += ":limit:" + limit
        }
        return key
    },
}
```

### Conditional Caching

```go
opts := &cache.CachedHandlerOptions{
    ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
        // Don't cache if bypass header present
        if c.GetHeader("X-Cache-Bypass") == "true" {
            return false
        }

        // Don't cache empty feeds
        if feed == nil || len(feed.Items) == 0 {
            return false
        }

        // Don't cache errors
        if c.Query("error") != "" {
            return false
        }

        // Don't cache if request has special flags
        if c.Query("nocache") == "1" {
            return false
        }

        return true
    },
}
```

### Multi-tenant Caching

```go
opts := &cache.CachedHandlerOptions{
    KeyGenerator: func(c *gin.Context) string {
        tenant := c.GetHeader("X-Tenant-ID")
        path := c.Request.URL.Path
        return "tenant:" + tenant + ":" + path
    },
    ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
        // Different caching rules per tenant
        tenant := c.GetHeader("X-Tenant-ID")
        if tenant == "premium" {
            return true // Cache for premium users
        }
        return len(feed.Items) > 10 // Only cache if enough items
    },
}
```

---

## Testing Cached Handlers

```go
func TestCachedHandler(t *testing.T) {
    gin.SetMode(gin.TestMode)

    cacheInstance := cache.NewMemoryCache(100)
    callCount := 0

    handler := cache.NewCachedHandler(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
        callCount++
        return &models.Feed{Title: "Test Feed"}, nil
    })

    router := gin.New()
    router.GET("/test", handler)

    // First request - cache miss
    w1 := httptest.NewRecorder()
    req1, _ := http.NewRequest("GET", "/test", nil)
    router.ServeHTTP(w1, req1)

    assert.Equal(t, http.StatusOK, w1.Code)
    assert.Equal(t, "MISS", w1.Header().Get("X-Cache"))
    assert.Equal(t, 1, callCount)

    // Second request - cache hit
    w2 := httptest.NewRecorder()
    req2, _ := http.NewRequest("GET", "/test", nil)
    router.ServeHTTP(w2, req2)

    assert.Equal(t, http.StatusOK, w2.Code)
    assert.Equal(t, "HIT", w2.Header().Get("X-Cache"))
    assert.Equal(t, 1, callCount) // Handler not called again
}
```

---

## Troubleshooting

### Cache Not Working

1. **Check bypass list**: Ensure your path isn't in the bypass list
2. **Check feed is not nil**: Handler must return a valid feed
3. **Check cache instance**: Ensure cacheInstance is properly initialized
4. **Check X-Cache header**: "MISS" means caching attempted, empty means bypassed

### High Memory Usage

1. **Reduce TTL**: Shorter cache duration means fewer items in memory
2. **Use Redis**: Switch from memory cache to Redis for distributed caching
3. **Limit cache size**: Set maximum cache size in LRU cache

### Stale Content

1. **Reduce TTL**: Faster expiration means fresher content
2. **Implement manual invalidation**: Trigger cache refresh on content updates
3. **Use cache warming**: Pre-populate cache with scheduled jobs

---

## Performance Tips

1. **Use appropriate TTLs**:
   - Fast-changing data: 1-5 minutes
   - Medium frequency: 15-30 minutes
   - Slow-changing: 1-24 hours

2. **Cache only successful responses**:
   ```go
   ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
       return feed != nil && len(feed.Items) > 0
   }
   ```

3. **Use ETags** to reduce bandwidth:
   ```go
   opts := &cache.CachedHandlerOptions{
       ETagEnabled: true,
   }
   ```

4. **Monitor cache hit ratio**:
   - Hit ratio > 80% = excellent
   - Hit ratio > 60% = good
   - Hit ratio < 40% = consider tuning TTL

---

## See Also

- [examples/handler_caching_example.go](../examples/handler_caching_example.go) - More code examples
- [internal/cache/handler.go](../internal/cache/handler.go) - Implementation
- [internal/middleware/cache.go](../internal/middleware/cache.go) - Middleware implementation
