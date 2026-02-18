package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// Header keys
	headerETag             = "Etag"
	headerCacheControl     = "Cache-Control"
	headerXContentTypeOpts = "X-Content-Type-Options"
	headerXRSSHubRoute     = "X-RSSHub-Route"

	// CORS headers
	headerAccessControlAllowOrigin  = "Access-Control-Allow-Origin"
	headerAccessControlAllowMethods = "Access-Control-Allow-Methods"
	headerAccessControlAllowHeaders = "Access-Control-Allow-Headers"
	headerAccessControlMaxAge       = "Access-Control-Max-Age"

	// Cache status header
	headerRSSHubCacheStatus = "RSSHub-Cache-Status"
)

// Header returns a middleware that sets HTTP headers
func Header(cacheTTL time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle OPTIONS requests for CORS preflight
		if c.Request.Method == http.MethodOptions {
			c.Header(headerAccessControlAllowOrigin, "*")
			c.Header(headerAccessControlAllowMethods, "GET, POST, PUT, DELETE, OPTIONS")
			c.Header(headerAccessControlAllowHeaders, "Content-Type, Authorization")
			c.Header(headerAccessControlMaxAge, "86400")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// Set CORS headers
		c.Header(headerAccessControlAllowOrigin, "*")

		// Set security headers
		c.Header(headerXContentTypeOpts, "nosniff")

		// Set cache headers
		if cacheTTL > 0 {
			maxAge := int(cacheTTL.Seconds())
			c.Header(headerCacheControl, fmt.Sprintf("public, max-age=%d", maxAge))
		} else {
			c.Header(headerCacheControl, "no-cache")
		}

		// Set RSSHub route header
		if route := c.Request.URL.Path; route != "" && route != "/" {
			c.Header(headerXRSSHubRoute, route)
		}

		// Check for If-None-Match header (ETag support)
		if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" {
			c.Set("if-none-match", ifNoneMatch)
		}

		c.Next()
	}
}

// setETag calculates and sets ETag header from response body
// Returns true if ETag was set and matches (304 should be returned)
func setETag(c *gin.Context, body []byte) bool {
	if len(body) == 0 {
		return false
	}

	// Generate ETag from body
	etag := generateETagFromBody(body)
	c.Header(headerETag, etag)

	// Check if client has cached version
	if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" {
		// Trim whitespace and quotes for comparison
		clientETag := strings.Trim(strings.TrimSpace(ifNoneMatch), `"`)
		serverETag := strings.Trim(etag, `"`)

		if clientETag == serverETag || ifNoneMatch == etag {
			return true // ETag matches, return 304
		}
	}

	return false
}

// setCacheStatus sets the RSSHub-Cache-Status header
func setCacheStatus(c *gin.Context, status string) {
	c.Header(headerRSSHubCacheStatus, status)
}

// setCacheHeaders sets cache-related headers
func setCacheHeaders(c *gin.Context, ttl time.Duration) {
	if ttl > 0 {
		maxAge := int(ttl.Seconds())
		c.Header(headerCacheControl, fmt.Sprintf("public, max-age=%d", maxAge))
	} else {
		c.Header(headerCacheControl, "no-cache")
	}
}
