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
	"go.uber.org/zap"
	rssubcache "github.com/rsshub/go/pkg/cache"
	"github.com/rsshub/go/pkg/logger"
	"github.com/rsshub/go/pkg/models"
)

// HandlerFunc is the function signature for cached handlers
type HandlerFunc func(*gin.Context) (*models.Feed, error)

// CachedHandlerOptions configures caching behavior
type CachedHandlerOptions struct {
	TTL           time.Duration
	KeyGenerator  func(*gin.Context) string
	ShouldCache   func(*gin.Context, *models.Feed) bool
	ETagEnabled   bool
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

		// Debug: log cache lookup
		if logger.Logger.Core().Enabled(zap.DebugLevel) {
			logger.Logger.Debug("[CACHE] Lookup",
				zap.String("key", cacheKey),
				zap.String("path", c.Request.URL.Path),
				zap.String("query", c.Request.URL.RawQuery))
		}

		// Check if we have a cached response
		if cached, ok := cacheInstance.Get(cacheKey); ok {
			if cachedResp, ok := cached.(*CachedResponse); ok {
				if logger.Logger.Core().Enabled(zap.DebugLevel) {
					logger.Logger.Debug("[CACHE] HIT",
						zap.String("key", cacheKey),
						zap.Int("status", cachedResp.StatusCode),
						zap.Int("body_len", len(cachedResp.Body)))
				}
				// Check ETag for 304 response
				if opts.ETagEnabled {
					if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" {
						if ifNoneMatch == cachedResp.ETag {
							c.Header("ETag", cachedResp.ETag)
							c.Header("X-Cache", "HIT")
							c.AbortWithStatus(http.StatusNotModified)
							return
						}
					}
					c.Header("ETag", cachedResp.ETag)
				}

				// Return cached response
				c.Header("Content-Type", cachedResp.ContentType)
				c.Header("X-Cache", "HIT")
				c.Data(cachedResp.StatusCode, cachedResp.ContentType, cachedResp.Body)
				c.Abort()
				return
			}
		}

		if logger.Logger.Core().Enabled(zap.DebugLevel) {
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

		// Check if we should cache this response
		if opts.ShouldCache(c, feed) {
			// Serialize feed to response format
			format := c.DefaultQuery("format", "rss")
			var contentType string
			var body []byte

			switch format {
			case "atom":
				body, contentType = serializeAtom(feed)
			case "json":
				body, contentType = serializeJSON(feed)
			default:
				body, contentType = serializeRSS(feed)
			}

			// Generate ETag
			var etag string
			if opts.ETagEnabled {
				etag = generateETag(body)
				c.Header("ETag", etag)
			}

			// Set cache status
			c.Header("X-Cache", "MISS")
			c.Header("Content-Type", contentType)

			// Store in cache
			cachedResp := &CachedResponse{
				StatusCode:  http.StatusOK,
				ContentType: contentType,
				Body:        body,
				ETag:        etag,
				ExpiresAt:   time.Now().Add(opts.TTL),
			}

			if logger.Logger.Core().Enabled(zap.DebugLevel) {
				logger.Logger.Debug("[CACHE] SET",
					zap.String("key", cacheKey),
					zap.Duration("ttl", opts.TTL),
					zap.Int("items", len(feed.Items)),
					zap.Int("body_len", len(body)))
			}

			cacheInstance.Set(cacheKey, cachedResp, opts.TTL)

			// Return response
			c.Data(http.StatusOK, contentType, body)
		} else {
			if logger.Logger.Core().Enabled(zap.DebugLevel) {
				logger.Logger.Debug("[CACHE] SKIP",
					zap.String("key", cacheKey),
					zap.String("reason", "ShouldCache returned false"))
			}
			// Not caching, just return the response
			format := c.DefaultQuery("format", "rss")
			var contentType string
			var body []byte

			switch format {
			case "atom":
				body, contentType = serializeAtom(feed)
			case "json":
				body, contentType = serializeJSON(feed)
			default:
				body, contentType = serializeRSS(feed)
			}

			c.Header("Content-Type", contentType)
			c.Data(http.StatusOK, contentType, body)
		}
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

// serializeRSS converts a feed to RSS format
func serializeRSS(feed *models.Feed) ([]byte, string) {
	// This would use the actual RSS generation code
	// For now, return a placeholder
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>%s</title>
    <description>%s</description>
    <link>%s</link>
    %s
  </channel>
</rss>`, feed.Title, feed.Description, feed.Link, serializeItems(feed.Items))

	return []byte(xml), "application/rss+xml; charset=utf-8"
}

// serializeAtom converts a feed to Atom format
func serializeAtom(feed *models.Feed) ([]byte, string) {
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>%s</title>
  <subtitle>%s</subtitle>
  <link href="%s"/>
  %s
</feed>`, feed.Title, feed.Description, feed.Link, serializeItemsAtom(feed.Items))

	return []byte(xml), "application/atom+xml; charset=utf-8"
}

// serializeJSON converts a feed to JSON format
func serializeJSON(feed *models.Feed) ([]byte, string) {
	json, err := json.MarshalIndent(feed, "", "  ")
	if err != nil {
		return []byte("{}"), "application/json; charset=utf-8"
	}
	return json, "application/json; charset=utf-8"
}

// serializeItems converts feed items to RSS item format
func serializeItems(items []models.Item) string {
	var result string
	for _, item := range items {
		result += fmt.Sprintf(`
    <item>
      <title>%s</title>
      <description>%s</description>
      <link>%s</link>
      <guid>%s</guid>
    </item>`, item.Title, item.Description, item.Link, item.GUID)
	}
	return result
}

// serializeItemsAtom converts feed items to Atom entry format
func serializeItemsAtom(items []models.Item) string {
	var result string
	for _, item := range items {
		result += fmt.Sprintf(`
  <entry>
    <title>%s</title>
    <content>%s</content>
    <link href="%s"/>
    <id>%s</id>
  </entry>`, item.Title, item.Description, item.Link, item.GUID)
	}
	return result
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
