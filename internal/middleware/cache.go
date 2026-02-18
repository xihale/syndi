package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xihale/rsshub-go/pkg/cache"
)

const (
	// Cache key prefixes
	cacheKeyPrefix     = "feed"
	requestingPrefix   = "requesting"
	cacheStatusHeader  = "RSSHub-Cache-Status"
	cacheBypassHeader  = "RSSHub-Cache-Bypass"
)

// Cache returns a middleware that caches responses
func Cache(cacheInstance cache.Cache, routeExpire time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if cache is bypassed via header
		if c.GetHeader(cacheBypassHeader) == "true" {
			c.Header(cacheStatusHeader, "BYPASS")
			c.Next()
			return
		}

		// Bypass list - paths that should never be cached
		path := c.Request.URL.Path
		if shouldBypassCache(path) {
			c.Next()
			return
		}

		// Generate cache key
		cacheKey := generateCacheKey(c)
		controlKey := requestingPrefix + ":" + cacheKey

		// Check cache for existing response
		if cached, ok := cacheInstance.Get(cacheKey); ok {
			if cachedResp, ok := cached.(*CachedResponse); ok {
				// Check ETag
				if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" {
					if ifNoneMatch == cachedResp.ETag {
						c.Header("ETag", cachedResp.ETag)
						c.Header(cacheStatusHeader, "HIT")
						c.AbortWithStatus(http.StatusNotModified)
						return
					}
				}

				// Return cached response
				c.Header("Content-Type", cachedResp.ContentType)
				c.Header("ETag", cachedResp.ETag)
				c.Header(cacheStatusHeader, "HIT")
				c.Data(cachedResp.StatusCode, cachedResp.ContentType, cachedResp.Body)
				c.Abort()
				return
			}
		}

		// Request deduplication - check if another request is in progress
		if isInProgress(cacheInstance, controlKey) {
			// Wait for the other request to complete
			if waitForCache(cacheInstance, cacheKey, controlKey, 10) {
				// Cache is now available, retry getting it
				if cached, ok := cacheInstance.Get(cacheKey); ok {
					if cachedResp, ok := cached.(*CachedResponse); ok {
						c.Header("Content-Type", cachedResp.ContentType)
						c.Header("ETag", cachedResp.ETag)
						c.Header(cacheStatusHeader, "HIT")
						c.Data(cachedResp.StatusCode, cachedResp.ContentType, cachedResp.Body)
						c.Abort()
						return
					}
				}
			}

			// Failed to get cached response, return 503
			c.Header(cacheStatusHeader, "WAIT_TIMEOUT")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Request in progress, please retry",
			})
			return
		}

		// Mark request as in progress
		setInProgress(cacheInstance, controlKey, 30*time.Second)
		defer cacheInstance.Delete(controlKey)

		// Wrap response writer to capture output
		wrapped := newResponseWriterWrapper(c.Writer)
		c.Writer = wrapped

		// Process request
		c.Next()

		// Get status code and body from wrapper
		statusCode := wrapped.Status()
		body := wrapped.Bytes()

		// Check if writer was reset (this shouldn't happen but let's handle it)
		// If c.Writer is no longer our wrapped writer, we can't cache
		if c.Writer != wrapped {
			// Writer was reset, can't properly cache
			return
		}

		// Generate ETag
		var etag string
		if statusCode >= 200 && statusCode < 300 && len(body) > 0 {
			etag = generateETagFromBody(body)
			c.Writer.Header().Set("ETag", etag)
		}

		// Set cache status header
		if statusCode >= 200 && statusCode < 300 {
			c.Writer.Header().Set(cacheStatusHeader, "MISS")
		} else if statusCode >= 400 {
			c.Writer.Header().Set(cacheStatusHeader, "ERROR")
		}

		// Flush buffered response to underlying writer
		wrapped.Flush()

		// Only cache successful responses
		if statusCode >= 200 && statusCode < 300 && len(body) > 0 {
			// Create cached response
			cachedResp := &CachedResponse{
				StatusCode:  statusCode,
				ContentType: c.Writer.Header().Get("Content-Type"),
				Body:        body,
				ETag:        etag,
			}

			// Store in cache
			cacheInstance.Set(cacheKey, cachedResp, routeExpire)
		}
	}
}

// CachedResponse represents a cached HTTP response
type CachedResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
	ETag        string
}

// shouldBypassCache checks if the path should bypass caching
func shouldBypassCache(path string) bool {
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

// generateCacheKey generates a cache key from the request
func generateCacheKey(c *gin.Context) string {
	path := c.Request.URL.Path
	format := c.Query("format")
	limit := c.Query("limit")

	// Build cache key: feed:path:format:limit
	key := fmt.Sprintf("%s:%s", cacheKeyPrefix, path)
	if format != "" {
		key += ":" + format
	}
	if limit != "" {
		key += ":" + limit
	}

	return key
}

// isInProgress checks if a request is already in progress
func isInProgress(cacheInstance cache.Cache, controlKey string) bool {
	return cacheInstance.Exists(controlKey)
}

// setInProgress marks a request as in progress
func setInProgress(cacheInstance cache.Cache, controlKey string, ttl time.Duration) {
	cacheInstance.Set(controlKey, "1", ttl)
}

// waitForCache waits for another request to populate the cache
func waitForCache(cacheInstance cache.Cache, cacheKey, controlKey string, maxRetries int) bool {
	for i := 0; i < maxRetries; i++ {
		time.Sleep(500 * time.Millisecond)

		// Check if request is still in progress
		if !cacheInstance.Exists(controlKey) {
			// Request completed, check cache
			if cacheInstance.Exists(cacheKey) {
				return true
			}
			// Request completed but cache not set (likely error)
			return false
		}
	}

	return false
}

// RequestCounter tracks concurrent requests per key
type RequestCounter struct {
	count atomic.Int32
}

// Increment increments the request counter
func (rc *RequestCounter) Increment() int32 {
	return rc.count.Add(1)
}

// Decrement decrements the request counter
func (rc *RequestCounter) Decrement() int32 {
	return rc.count.Add(-1)
}

// Get returns the current count
func (rc *RequestCounter) Get() int32 {
	return rc.count.Load()
}

// parseCacheTTL parses a cache TTL from a string
func parseCacheTTL(ttlStr string, defaultTTL time.Duration) time.Duration {
	if ttlStr == "" {
		return defaultTTL
	}

	ttl, err := strconv.ParseInt(ttlStr, 10, 64)
	if err != nil {
		return defaultTTL
	}

	return time.Duration(ttl) * time.Second
}

// parseMaxAge parses Cache-Control header to get max-age value
func parseMaxAge(cacheControl string) int {
	parts := strings.Split(cacheControl, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "max-age=") {
			ageStr := strings.TrimPrefix(part, "max-age=")
			if age, err := strconv.Atoi(ageStr); err == nil {
				return age
			}
		}
	}
	return 0
}
