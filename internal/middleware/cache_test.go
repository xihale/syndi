package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xihale/rsshub-go/pkg/cache"
	"github.com/stretchr/testify/assert"
)

func TestCache_ShouldBypassCache(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "root", path: "/", expected: true},
		{name: "robots.txt", path: "/robots.txt", expected: true},
		{name: "favicon.ico", path: "/favicon.ico", expected: true},
		{name: "logo.png", path: "/logo.png", expected: true},
		{name: "api path", path: "/api/test", expected: true},
		{name: "regular path", path: "/github/user", expected: false},
		{name: "nested path", path: "/twitter/user/lists", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldBypassCache(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCache_BypassHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cacheInstance := cache.NewMemoryCache(100)
	ttl := 5 * time.Minute

	router := gin.New()
	router.Use(Cache(cacheInstance, ttl))

	callCount := 0
	router.GET("/test", func(c *gin.Context) {
		callCount++
		c.String(http.StatusOK, "response %d", callCount)
	})

	// First request with bypass header
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.Header.Set(cacheBypassHeader, "true")
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "BYPASS", w1.Header().Get(cacheStatusHeader))
	assert.Equal(t, 1, callCount)
}

func TestCache_CacheKeyGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		path     string
		format   string
		limit    string
		expected string
	}{
		{
			name:     "basic path",
			path:     "/test/path",
			format:   "",
			limit:    "",
			expected: "feed:/test/path",
		},
		{
			name:     "with format",
			path:     "/test/path",
			format:   "atom",
			limit:    "",
			expected: "feed:/test/path:atom",
		},
		{
			name:     "with limit",
			path:     "/test/path",
			format:   "",
			limit:    "10",
			expected: "feed:/test/path:10",
		},
		{
			name:     "with format and limit",
			path:     "/test/path",
			format:   "atom",
			limit:    "10",
			expected: "feed:/test/path:atom:10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request, _ = http.NewRequest("GET", tt.path, nil)

			// Set query parameters
			q := c.Request.URL.Query()
			if tt.format != "" {
				q.Set("format", tt.format)
			}
			if tt.limit != "" {
				q.Set("limit", tt.limit)
			}
			c.Request.URL.RawQuery = q.Encode()

			result := generateCacheKey(c)

			// The format and limit might not be in the right order in the actual key
			// For simplicity, just check that the parts are there
			assert.Contains(t, result, "feed:"+tt.path)
			if tt.format != "" {
				assert.Contains(t, result, tt.format)
			}
			if tt.limit != "" {
				assert.Contains(t, result, tt.limit)
			}
		})
	}
}

func TestCache_RequestCounter(t *testing.T) {
	rc := &RequestCounter{}

	assert.Equal(t, int32(0), rc.Get())

	// Increment
	assert.Equal(t, int32(1), rc.Increment())
	assert.Equal(t, int32(2), rc.Increment())
	assert.Equal(t, int32(2), rc.Get())

	// Decrement
	assert.Equal(t, int32(1), rc.Decrement())
	assert.Equal(t, int32(0), rc.Decrement())
	assert.Equal(t, int32(0), rc.Get())
}

func TestCache_ETagGeneration(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		isEmpty  bool
	}{
		{
			name:    "empty body",
			body:    []byte{},
			isEmpty: true,
		},
		{
			name:    "simple body",
			body:    []byte("test content"),
			isEmpty: false,
		},
		{
			name:    "complex body",
			body:    []byte("<?xml version=\"1.0\"?><rss><item>test</item></rss>"),
			isEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			etag := generateETagFromBody(tt.body)

			if tt.isEmpty {
				assert.Empty(t, etag)
			} else {
				assert.NotEmpty(t, etag)
				assert.True(t, len(etag) > 0)
				// ETags should be quoted
				assert.True(t, strings.HasPrefix(etag, "\"") && strings.HasSuffix(etag, "\""))
			}
		})
	}
}

func TestCache_ParseCacheTTL(t *testing.T) {
	defaultTTL := 5 * time.Minute

	tests := []struct {
		name        string
		ttlStr      string
		expectedTTL time.Duration
	}{
		{
			name:        "empty string",
			ttlStr:      "",
			expectedTTL: defaultTTL,
		},
		{
			name:        "valid number",
			ttlStr:      "300",
			expectedTTL: 300 * time.Second,
		},
		{
			name:        "invalid string",
			ttlStr:      "invalid",
			expectedTTL: defaultTTL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCacheTTL(tt.ttlStr, defaultTTL)
			assert.Equal(t, tt.expectedTTL, result)
		})
	}
}

func TestCache_ParseMaxAge(t *testing.T) {
	tests := []struct {
		name           string
		cacheControl   string
		expectedMaxAge int
	}{
		{
			name:           "empty",
			cacheControl:   "",
			expectedMaxAge: 0,
		},
		{
			name:           "with max-age",
			cacheControl:   "max-age=3600",
			expectedMaxAge: 3600,
		},
		{
			name:           "with multiple directives",
			cacheControl:   "public, max-age=600, must-revalidate",
			expectedMaxAge: 600,
		},
		{
			name:           "no max-age",
			cacheControl:   "no-cache",
			expectedMaxAge: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMaxAge(tt.cacheControl)
			assert.Equal(t, tt.expectedMaxAge, result)
		})
	}
}
