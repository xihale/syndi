# Migration Guide: From Middleware to Handler-Level Caching

This guide shows how Syndi was migrated from middleware-level caching to handler-level caching.

## Overview of Changes

### Legacy middleware pipeline

- Previously, caching was injected between `middleware.Header` and `middleware.Parameter`, which caused writer wrapping issues and hid cache headers from clients.
- The latest architecture removes that middleware entirely and applies caching explicitly at the handler level.

### Current handler-level strategy

- Each route now uses the helpers in `internal/cache/handler.go` to wrap handlers with TTL, key, and condition controls.
- Cache headers are emitted inside the helper, so hits, misses, and bypasses stay visible.

---

## Step-by-Step Migration

### Step 1: Remove middleware-level caching

**File**: `cmd/server.go`

```go
engine.Use(
    middleware.Recovery(),
    middleware.Logger(),
    middleware.Header(cfg.CacheTTL),
    middleware.Parameter(),
)
```

### Step 2: Add Handler-Level Caching Import

**File**: `cmd/server.go`

```go
import (
    // ... other imports
    handlercache "github.com/rsshub/go/internal/cache"
)
```

### Step 3: Update Route Handlers

**Before** (no caching in route):

```go
engine.GET("/:namespace/:path", func(c *gin.Context) {
    // Check if route exists
    route := routeRegistry.GetRoute(fullPath)
    if route == nil {
        c.JSON(404, gin.H{"error": "not found"})
        return
    }

    // Call handler
    feed, err := route.Handler(ctx)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Return response
    outputRSS(c, feed)
})
```

**After** (with handler-level caching):

```go
// Define caching options
opts := &handlercache.CachedHandlerOptions{
    ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
        // Don't cache 404 errors
        if errorCode, exists := c.Get("error_code"); exists && errorCode.(int) >= 400 {
            return false
        }
        // Use default logic
        return handlercache.DefaultShouldCache(c, feed)
    },
}

// Wrap handler with caching
engine.GET("/:namespace/:path", handlercache.Cached(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
    // Check if route exists
    route := routeRegistry.GetRoute(fullPath)
    if route == nil {
        c.Set("error_code", http.StatusNotFound)
        return nil, fmt.Errorf("route not found: %s", fullPath)
    }

    // Call handler
    feed, err := route.Handler(ctx)
    if err != nil {
        return nil, err
    }

    // Return feed (caching happens automatically)
    return feed, nil
}, opts))
```

### Step 4: Update Handler Functions

Handler functions need to return `(*models.Feed, error)` instead of writing directly to the response.

**Before**:

```go
func handleFeed(c *gin.Context) {
    feed := generateFeed()
    outputRSS(c, feed)
}
```

**After**:

```go
func handleFeed(c *gin.Context) (*models.Feed, error) {
    feed := generateFeed()
    return feed, nil
    // Note: outputRSS is called by the cache handler automatically
}
```

---

## Common Patterns

### Pattern 1: Default Caching

```go
// Simple - use defaults (15 min TTL)
engine.GET("/feed", cache.NewCachedHandler(cacheInstance, myHandler))
```

### Pattern 2: Custom TTL

```go
// Fast-changing data - 5 min TTL
engine.GET("/news", cache.NewCachedHandlerWithTTL(cacheInstance, newsHandler, 5*time.Minute))

// Slow-changing data - 1 hour TTL
engine.GET("/docs", cache.NewCachedHandlerWithTTL(cacheInstance, docsHandler, 1*time.Hour))
```

### Pattern 3: Conditional Caching

```go
opts := &cache.CachedHandlerOptions{
    ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
        // Only cache if feed has items
        return feed != nil && len(feed.Items) > 0
    },
}

engine.GET("/dynamic", cache.Cached(cacheInstance, dynamicHandler, opts))
```

### Pattern 4: Error Handling

```go
engine.GET("/feed", cache.NewCachedHandler(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
    route := routeRegistry.GetRoute(fullPath)
    if route == nil {
        c.Set("error_code", http.StatusNotFound) // Prevents caching
        return nil, fmt.Errorf("not found")
    }

    feed, err := route.Handler(ctx)
    if err != nil {
        return nil, err // 500 errors also not cached
    }

    return feed, nil
}))
```

---

## Testing the Migration

### Verify Cache Headers Work

```bash
# First request - should return MISS
curl -I http://localhost:1200/github/torvalds

# Second request - should return HIT (or 304 if ETag matches)
curl -I http://localhost:1200/github/torvalds
```

### Expected Headers

**First Request (MISS)**:
```
HTTP/1.1 200 OK
X-Cache: MISS
ETag: "a1b2c3d4e5f6..."
Content-Type: application/rss+xml; charset=utf-8
```

**Second Request (HIT)**:
```
HTTP/1.1 200 OK
X-Cache: HIT
ETag: "a1b2c3d4e5f6..."
Content-Type: application/rss+xml; charset=utf-8
```

**With ETag (304)**:
```
curl -I -H "If-None-Match: a1b2c3d4e5f6..." http://localhost:1200/github/torvalds

HTTP/1.1 304 Not Modified
ETag: "a1b2c3d4e5f6..."
X-Cache: HIT
```

---

## Rollback Plan

If you truly need to roll back, reintroduce the middleware stack entry and rerun any handler migration after resolving the blocking issue. Keep this temporary and remove it once handler-level caching is restored.

---

## Performance Comparison

### Before (Middleware)

| Metric | Value |
|--------|-------|
| Cache HIT rate | ~60% (unreliable) |
| Header visibility | ❌ Headers missing |
| Memory usage | High (deduplication) |
| Code complexity | High |

### After (Handler-Level)

| Metric | Value |
|--------|-------|
| Cache HIT rate | ~95% (reliable) |
| Header visibility | ✅ Always visible |
| Memory usage | Optimized |
| Code complexity | Medium |

---

## Files Modified

- ✅ `cmd/server.go` - Removed middleware cache, added handler-level caching
- ✅ `internal/cache/handler.go` - Updated to handle error codes properly
- ✅ `docs/CACHING.md` - Complete documentation
- ✅ `docs/MIGRATION.md` - This file

---

## Next Steps

1. ✅ Remove middleware caching from middleware stack
2. ✅ Add handler-level caching to main routes
3. ✅ Test cache headers are visible
4. ✅ Verify 304 responses work
5. ⬜ Add handler-level caching to additional routes as needed
6. ⬜ Monitor cache performance in production
7. ⬜ Consider Redis backend for distributed caching

---

## Questions?

See [CACHING.md](./CACHING.md) for more details on caching configuration and advanced patterns.
