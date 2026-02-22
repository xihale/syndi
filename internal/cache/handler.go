package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xihale/rsshub-go/internal/middleware"
	rssubcache "github.com/xihale/rsshub-go/pkg/cache"
	"github.com/xihale/rsshub-go/pkg/logger"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/rss"
	"go.uber.org/zap"
)

// HandlerFunc is the function signature for cached handlers
type HandlerFunc func(*gin.Context) (*models.Feed, error)

// CachedHandlerOptions configures caching behavior
type CachedHandlerOptions struct {
	TTL          time.Duration
	KeyGenerator func(*gin.Context) string
	ShouldCache  func(*gin.Context, *models.Feed) bool
	ETagEnabled  bool
}

// DefaultCachedOptions returns default caching options
func DefaultCachedOptions() *CachedHandlerOptions {
	return &CachedHandlerOptions{
		TTL:          15 * time.Minute,
		KeyGenerator: DefaultKeyGenerator,
		ShouldCache:  DefaultShouldCache,
		ETagEnabled:  true,
	}
}

// Cached wraps a handler with caching logic
// It returns a gin.HandlerFunc that can be used directly in routes
func Cached(cacheInstance rssubcache.Cache, handler HandlerFunc, opts *CachedHandlerOptions) gin.HandlerFunc {
	if opts == nil {
		opts = DefaultCachedOptions()
	}

	return func(c *gin.Context) {
		// Generate cache key
		cacheKey := opts.KeyGenerator(c)

		// Debug: log cache lookup (only if logger is initialized)
		if logger.Logger != nil && logger.Logger.Core() != nil && logger.Logger.Core().Enabled(zap.DebugLevel) {
			logger.Logger.Debug("[CACHE] Lookup",
				zap.String("key", cacheKey),
				zap.String("path", c.Request.URL.Path),
				zap.String("query", c.Request.URL.RawQuery))
		}

		// Check if we have a cached full feed (*models.Feed) - the RAW feed without parameter processing
		if cachedFeed, ok := cacheInstance.Get(cacheKey); ok {
			// Since we use gob serialization, the cached value should already be *models.Feed
			feed, ok := cachedFeed.(*models.Feed)
			if !ok {
				// Type assertion failed - treat as cache miss
				if logger.Logger != nil && logger.Logger.Core() != nil && logger.Logger.Core().Enabled(zap.DebugLevel) {
					logger.Logger.Debug("[CACHE] TYPE MISMATCH",
						zap.String("key", cacheKey),
						zap.String("expected", "*models.Feed"),
						zap.Any("got", cachedFeed))
				}
			} else if feed != nil {
				if logger.Logger != nil && logger.Logger.Core() != nil && logger.Logger.Core().Enabled(zap.DebugLevel) {
					logger.Logger.Debug("[CACHE] HIT",
						zap.String("key", cacheKey),
						zap.Int("raw_items", len(feed.Items)))
				}
				// Apply query parameter processing to the cached raw feed
				processedItems := middleware.ProcessFeed(c, feed.Items)

				// Build response feed with processed items
				responseFeed := &models.Feed{
					Title:       feed.Title,
					Link:        feed.Link,
					Description: feed.Description,
					Items:       processedItems,
				}

				// Serialize to response format
				format := c.DefaultQuery("format", "rss")
				var contentType string
				var body []byte
				var err error

				body, contentType, err = serializeFeed(responseFeed, format)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				// Set cache status header
				c.Header("X-Cache", "HIT")

				// Handle ETag
				if opts.ETagEnabled {
					etag := generateETag(body)
					c.Header("ETag", etag)

					if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" {
						if ifNoneMatch == etag {
							c.AbortWithStatus(http.StatusNotModified)
							return
						}
					}
				}

				c.Header("Content-Type", contentType)
				c.Data(http.StatusOK, contentType, body)
				c.Abort()
				return
			}
		}

		if logger.Logger != nil && logger.Logger.Core() != nil && logger.Logger.Core().Enabled(zap.DebugLevel) {
			logger.Logger.Debug("[CACHE] MISS",
				zap.String("key", cacheKey),
				zap.String("path", c.Request.URL.Path))
		}

		// Call the actual handler
		feed, err := handler(c)
		if err != nil {
			// Check for custom error code
			if errorCode, exists := c.Get("error_code"); exists {
				if code, ok := errorCode.(int); ok {
					c.JSON(code, gin.H{"error": err.Error()})
					return
				}
			}
			// Default to 500 for other errors
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Cache miss: fetch fresh feed from handler
		// feed was already retrieved from handler(c) above
		// Now we need to:
		// 1. Cache the RAW full feed for future requests
		// 2. Apply parameter processing for this request's response

		// Step 1: Cache raw full feed (if applicable)
		if opts.ShouldCache(c, feed) {
			// Important: store the unfiltered full feed
			if logger.Logger != nil && logger.Logger.Core() != nil && logger.Logger.Core().Enabled(zap.DebugLevel) {
				logger.Logger.Debug("[CACHE] SET",
					zap.String("key", cacheKey),
					zap.Duration("ttl", opts.TTL),
					zap.Int("raw_items", len(feed.Items)))
			}
			cacheInstance.Set(cacheKey, feed, opts.TTL)
		}

		// Step 2: Apply parameter processing for this specific response
		processedItems := middleware.ProcessFeed(c, feed.Items)
		responseFeed := &models.Feed{
			Title:       feed.Title,
			Link:        feed.Link,
			Description: feed.Description,
			Items:       processedItems,
		}

		// Serialize processed feed to response format
		format := c.DefaultQuery("format", "rss")
		var contentType string
		var body []byte

		body, contentType, err = serializeFeed(responseFeed, format)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Set cache status header
		c.Header("X-Cache", "MISS")

		// Handle ETag
		if opts.ETagEnabled {
			etag := generateETag(body)
			c.Header("ETag", etag)

			if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" {
				if ifNoneMatch == etag {
					c.AbortWithStatus(http.StatusNotModified)
					return
				}
			}
		}

		c.Header("Content-Type", contentType)
		c.Data(http.StatusOK, contentType, body)
	}
}

// CachedResponse represents a cached HTTP response
type CachedResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
	ETag        string
	ExpiresAt   time.Time
}

// DefaultKeyGenerator generates a cache key from the request context
func DefaultKeyGenerator(c *gin.Context) string {
	path := c.Request.URL.Path
	format := c.Query("format")
	// Note: we intentionally exclude 'limit' from cache key so that
	// different limit requests can share the same full feed cache

	// Build cache key: feed:path:format
	key := fmt.Sprintf("feed:%s", path)
	if format != "" {
		key += ":" + format
	}

	return key
}

// DefaultShouldCache determines if a response should be cached
func DefaultShouldCache(c *gin.Context, feed *models.Feed) bool {
	// Don't cache if explicitly bypassed
	if c.GetHeader("X-Cache-Bypass") == "true" {
		return false
	}

	// Don't cache if feed is nil or empty
	if feed == nil {
		return false
	}

	// Don't cache paths that shouldn't be cached
	path := c.Request.URL.Path
	if shouldBypassPath(path) {
		return false
	}

	// Cache all other successful responses
	return true
}

// shouldBypassPath checks if a path should bypass caching
func shouldBypassPath(path string) bool {
	// Exact match for root
	if path == "/" {
		return true
	}

	bypassList := []string{
		"/robots.txt",
		"/favicon.ico",
		"/logo.png",
		"/api/",
	}

	for _, bypass := range bypassList {
		if strings.HasPrefix(path, bypass) {
			return true
		}
	}

	return false
}

// generateETag generates an ETag from response body
func generateETag(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	hash := sha256.Sum256(body)
	return `"` + hex.EncodeToString(hash[:]) + `"`
}

// serializeFeed serializes feed output by requested format.
func serializeFeed(feed *models.Feed, format string) ([]byte, string, error) {
	switch format {
	case "atom":
		body, err := rss.GenerateAtom(feed)
		if err != nil {
			return nil, "", err
		}
		return body, "application/atom+xml; charset=utf-8", nil
	case "json":
		return serializeJSON(feed)
	default:
		body, err := rss.GenerateRSS(feed)
		if err != nil {
			return nil, "", err
		}
		return body, "application/rss+xml; charset=utf-8", nil
	}
}

// serializeJSON converts a feed to JSON format.
func serializeJSON(feed *models.Feed) ([]byte, string, error) {
	json, err := json.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return json, "application/json; charset=utf-8", nil
}

// NewCachedHandler creates a cached handler with default options
func NewCachedHandler(cacheInstance rssubcache.Cache, handler HandlerFunc) gin.HandlerFunc {
	return Cached(cacheInstance, handler, DefaultCachedOptions())
}

// NewCachedHandlerWithTTL creates a cached handler with a specific TTL
func NewCachedHandlerWithTTL(cacheInstance rssubcache.Cache, handler HandlerFunc, ttl time.Duration) gin.HandlerFunc {
	opts := DefaultCachedOptions()
	opts.TTL = ttl
	return Cached(cacheInstance, handler, opts)
}

// InvalidateCache removes a specific key from the cache
func InvalidateCache(cacheInstance rssubcache.Cache, c *gin.Context) {
	key := DefaultKeyGenerator(c)
	cacheInstance.Delete(key)
}

// InvalidateCacheByPattern removes all cache keys matching a pattern
func InvalidateCacheByPattern(cacheInstance rssubcache.Cache, pattern string) {
	// This would require cache pattern matching support
	// For now, it's a placeholder for future implementation
}

// GetCacheStats returns statistics about the cache
type CacheStats struct {
	Keys     int
	Hits     int64
	Misses   int64
	HitRatio float64
}

// GetCacheStats returns cache statistics (placeholder for future implementation)
func GetCacheStats(cacheInstance rssubcache.Cache) (*CacheStats, error) {
	// This would require the cache to support statistics tracking
	// For now, return a placeholder
	return &CacheStats{
		Keys:     0,
		Hits:     0,
		Misses:   0,
		HitRatio: 0.0,
	}, nil
}
