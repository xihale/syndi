package examples

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rsshub/go/internal/cache"
	rsshubcache "github.com/rsshub/go/pkg/cache"
	"github.com/rsshub/go/pkg/models"
	ctxpkg "github.com/rsshub/go/pkg/context"
)

// Example demonstrates handler-level caching with route handlers

// registerCachedRoute registers a route with handler-level caching
// This is an alternative to middleware-level caching
func registerCachedRoute(engine *gin.Engine, routePath string, routeHandler func(*ctxpkg.Context) (*models.Feed, error), cacheInstance rsshubcache.Cache) {
	// Wrap the route handler with caching
	cachedHandler := cache.NewCachedHandler(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
		// Call the original route handler
		ctx := ctxpkg.NewContext(c.Writer, c.Request)
		return routeHandler(ctx)
	})

	// Register the cached handler
	engine.GET(routePath, cachedHandler)
}

// registerCachedRouteWithCustomTTL registers a route with custom cache TTL
func registerCachedRouteWithCustomTTL(engine *gin.Engine, routePath string, routeHandler func(*ctxpkg.Context) (*models.Feed, error), cacheInstance rsshubcache.Cache, ttl time.Duration) {
	cachedHandler := cache.NewCachedHandlerWithTTL(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
		ctx := ctxpkg.NewContext(c.Writer, c.Request)
		return routeHandler(ctx)
	}, ttl)

	engine.GET(routePath, cachedHandler)
}

// registerCachedRouteWithOptions registers a route with custom caching options
func registerCachedRouteWithOptions(engine *gin.Engine, routePath string, routeHandler func(*ctxpkg.Context) (*models.Feed, error), cacheInstance rsshubcache.Cache, opts *cache.CachedHandlerOptions) {
	cachedHandler := cache.Cached(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
		ctx := ctxpkg.NewContext(c.Writer, c.Request)
		return routeHandler(ctx)
	}, opts)

	engine.GET(routePath, cachedHandler)
}

// Example usage in cmd/server.go:
/*
// Instead of:
engine.GET("/:namespace/:path", func(c *gin.Context) {
    // ... handler logic
})

// Use:
registerCachedRoute(engine, "/:namespace/:path", func(ctx *ctxpkg.Context) (*models.Feed, error) {
    // ... handler logic
}, cacheInstance)
*/

// Advanced example with custom key generator
func registerCachedRouteWithKeyGenerator(engine *gin.Engine, routePath string, routeHandler func(*ctxpkg.Context) (*models.Feed, error), cacheInstance rsshubcache.Cache) {
	opts := &cache.CachedHandlerOptions{
		TTL:     30 * time.Minute,
		ETagEnabled: true,
		KeyGenerator: func(c *gin.Context) string {
			// Custom key generation that includes user ID if present
			key := "feed:" + c.Request.URL.Path
			if userID := c.Query("user"); userID != "" {
				key += ":user:" + userID
			}
			return key
		},
		ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
			// Custom caching logic - only cache if feed has items
			return feed != nil && len(feed.Items) > 0
		},
	}

	cachedHandler := cache.Cached(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
		ctx := ctxpkg.NewContext(c.Writer, c.Request)
		return routeHandler(ctx)
	}, opts)

	engine.GET(routePath, cachedHandler)
}

// Example: Conditional caching based on response
func conditionalCacheExample(c *gin.Context, feed *models.Feed) bool {
	// Only cache if feed is not nil and has items
	if feed == nil {
		return false
	}

	// Don't cache empty feeds
	if len(feed.Items) == 0 {
		return false
	}

	// Don't cache if explicitly bypassed
	if c.GetHeader("X-Cache-Bypass") == "true" {
		return false
	}

	return true
}

// Example: Custom cache key for multi-tenant scenarios
func multiTenantKeyGenerator(c *gin.Context) string {
	path := c.Request.URL.Path
	format := c.Query("format")
	tenant := c.GetHeader("X-Tenant-ID")

	// Build key: feed:tenant:path:format
	key := "feed"
	if tenant != "" {
		key += ":" + tenant
	}
	key += ":" + path
	if format != "" {
		key += ":" + format
	}

	return key
}

// Example: Per-route cache configuration
type RouteCacheConfig struct {
	Enabled     bool
	TTL         time.Duration
	KeyPrefix   string
	Conditional func(*gin.Context, *models.Feed) bool
}

// Example configurations for different route types
var cacheConfigs = map[string]RouteCacheConfig{
	// Fast-changing data - short TTL
	"news": {
		Enabled: true,
		TTL:     5 * time.Minute,
	},
	// Slow-changing data - long TTL
	"documentation": {
		Enabled: true,
		TTL:     1 * time.Hour,
	},
	// User-specific data - custom key
	"user": {
		Enabled: true,
		TTL:     30 * time.Minute,
		KeyPrefix: "user",
	},
}

// Example: Setup multiple routes with different cache configs
func setupRoutesWithCacheConfigs(engine *gin.Engine, cacheInstance rsshubcache.Cache, routeRegistry interface{}) {
	// News route with short cache
	engine.GET("/news/:category", cache.NewCachedHandlerWithTTL(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
		// Handler implementation
		return &models.Feed{Title: "News"}, nil
	}, 5*time.Minute))

	// Documentation route with long cache
	engine.GET("/docs/:page", cache.NewCachedHandlerWithTTL(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
		return &models.Feed{Title: "Documentation"}, nil
	}, 1*time.Hour))

	// User-specific route with custom key
	opts := &cache.CachedHandlerOptions{
		TTL:     30 * time.Minute,
		KeyGenerator: multiTenantKeyGenerator,
		ShouldCache: conditionalCacheExample,
	}

	engine.GET("/user/:id/feed", cache.Cached(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
		return &models.Feed{Title: "User Feed"}, nil
	}, opts))
}
