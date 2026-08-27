package handlercache

import (
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/xihale/syndi/pkg/cache"
	"github.com/xihale/syndi/pkg/models"
)

func TestCached_CacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gob.Register(&models.Feed{})
	gob.Register(&models.Item{})
	gob.Register(&models.Author{})

	cacheInstance := cache.NewMemoryCache(100)
	callCount := 0

	handler := func(c *gin.Context) (*models.Feed, error) {
		callCount++
		return &models.Feed{
			Title:       "Test Feed",
			Description: "Test Description",
			Link:        "https://example.com",
			Items: []models.Item{
				{
					Title:   "Item 1",
					Link:    "https://example.com/1",
					GUID:    "1",
					PubDate: time.Now(),
				},
			},
		}, nil
	}

	cachedHandler := NewCachedHandler(cacheInstance, handler)

	router := gin.New()
	router.GET("/test", cachedHandler)

	// First request - cache miss
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "MISS", w1.Header().Get("X-Cache"))
	assert.Contains(t, w1.Body.String(), "Test Feed")
	assert.Equal(t, 1, callCount)

	// Verify the response was cached
	cacheKey := "feed:/test"
	_, exists := cacheInstance.Get(cacheKey)
	assert.True(t, exists, "Response should be cached")

	// Second request - cache hit
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "HIT", w2.Header().Get("X-Cache"))
	assert.Contains(t, w2.Body.String(), "Test Feed")
	assert.Equal(t, 1, callCount, "Handler should not be called again")
}

func TestCached_ETagSupport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gob.Register(&models.Feed{})
	gob.Register(&models.Item{})
	gob.Register(&models.Author{})

	cacheInstance := cache.NewMemoryCache(100)

	handler := func(c *gin.Context) (*models.Feed, error) {
		return &models.Feed{
			Title: "Test Feed",
			Items: []models.Item{
				{
					Title:   "Item 1",
					Link:    "https://example.com/1",
					GUID:    "1",
					PubDate: time.Now(),
				},
			},
		}, nil
	}

	cachedHandler := NewCachedHandler(cacheInstance, handler)

	router := gin.New()
	router.GET("/test", cachedHandler)

	// First request - get ETag
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w1, req1)

	etag := w1.Header().Get("ETag")
	assert.NotEmpty(t, etag)

	// Second request with If-None-Match
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("If-None-Match", etag)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusNotModified, w2.Code)
	assert.Equal(t, etag, w2.Header().Get("ETag"))
	assert.Equal(t, "HIT", w2.Header().Get("X-Cache"))
}

func TestCached_BypassHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cacheInstance := cache.NewMemoryCache(100)
	callCount := 0

	handler := func(c *gin.Context) (*models.Feed, error) {
		callCount++
		return &models.Feed{
			Title: "Test Feed",
		}, nil
	}

	cachedHandler := NewCachedHandler(cacheInstance, handler)

	router := gin.New()
	router.GET("/test", cachedHandler)

	// First request with bypass header
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-Cache-Bypass", "true")
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, 1, callCount)

	// Second request without bypass - should cache this time
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "MISS", w2.Header().Get("X-Cache"))
	assert.Equal(t, 2, callCount)
}

func TestCached_HandlerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cacheInstance := cache.NewMemoryCache(100)

	handler := func(c *gin.Context) (*models.Feed, error) {
		return nil, assert.AnError
	}

	cachedHandler := NewCachedHandler(cacheInstance, handler)

	router := gin.New()
	router.GET("/test", cachedHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestCached_FormatSupport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cacheInstance := cache.NewMemoryCache(100)

	handler := func(c *gin.Context) (*models.Feed, error) {
		return &models.Feed{
			Title: "Test Feed",
		}, nil
	}

	cachedHandler := NewCachedHandler(cacheInstance, handler)

	router := gin.New()
	router.GET("/test", cachedHandler)

	// Test RSS format (default)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "application/rss+xml; charset=utf-8", w1.Header().Get("Content-Type"))
	assert.Contains(t, w1.Body.String(), "<rss")

	// Test Atom format
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test?format=atom", nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "application/atom+xml; charset=utf-8", w2.Header().Get("Content-Type"))
	assert.Contains(t, w2.Body.String(), "<feed")

	// Test JSON format
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/test?format=json", nil)
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Equal(t, "application/json; charset=utf-8", w3.Header().Get("Content-Type"))
	assert.Contains(t, w3.Body.String(), "{")
}

func TestDefaultKeyGenerator(t *testing.T) {
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
			name:     "with format (format excluded: raw feed is format-independent)",
			path:     "/test/path",
			format:   "atom",
			limit:    "",
			expected: "feed:/test/path",
		},
		{
			name:     "with limit (limit is intentionally excluded from cache key)",
			path:     "/test/path",
			format:   "",
			limit:    "10",
			expected: "feed:/test/path",
		},
		{
			name:     "with format and limit (both intentionally excluded from cache key)",
			path:     "/test/path",
			format:   "atom",
			limit:    "10",
			expected: "feed:/test/path",
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

			result := DefaultKeyGenerator(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultShouldCache(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		path     string
		bypass   bool
		feed     *models.Feed
		expected bool
	}{
		{
			name:     "normal feed",
			path:     "/test/feed",
			bypass:   false,
			feed:     &models.Feed{Title: "Test"},
			expected: true,
		},
		{
			name:     "nil feed",
			path:     "/test/feed",
			bypass:   false,
			feed:     nil,
			expected: false,
		},
		{
			name:     "bypass header",
			path:     "/test/feed",
			bypass:   true,
			feed:     &models.Feed{Title: "Test"},
			expected: false,
		},
		{
			name:     "root path",
			path:     "/",
			bypass:   false,
			feed:     &models.Feed{Title: "Test"},
			expected: false,
		},
		{
			name:     "api path",
			path:     "/api/test",
			bypass:   false,
			feed:     &models.Feed{Title: "Test"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request, _ = http.NewRequest("GET", tt.path, nil)

			if tt.bypass {
				c.Request.Header.Set("X-Cache-Bypass", "true")
			}

			result := DefaultShouldCache(c, tt.feed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldBypassPath(t *testing.T) {
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
			result := shouldBypassPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateETag(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		isEmpty bool
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
			name:    "xml body",
			body:    []byte("<?xml version=\"1.0\"?><rss></rss>"),
			isEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			etag := generateETag(tt.body)

			if tt.isEmpty {
				assert.Empty(t, etag)
			} else {
				assert.NotEmpty(t, etag)
				// ETags should be quoted
				assert.True(t, len(etag) > 0)
			}
		})
	}
}

func TestNewCachedHandlerWithTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cacheInstance := cache.NewMemoryCache(100)
	customTTL := 30 * time.Second

	handler := func(c *gin.Context) (*models.Feed, error) {
		return &models.Feed{Title: "Test"}, nil
	}

	cachedHandler := NewCachedHandlerWithTTL(cacheInstance, handler, customTTL)

	router := gin.New()
	router.GET("/test", cachedHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "MISS", w.Header().Get("X-Cache"))
}

func TestInvalidateCache(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cacheInstance := cache.NewMemoryCache(100)

	handler := func(c *gin.Context) (*models.Feed, error) {
		return &models.Feed{Title: "Test"}, nil
	}

	cachedHandler := NewCachedHandler(cacheInstance, handler)

	router := gin.New()
	router.GET("/test", cachedHandler)

	// First request - cache the response
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w1, req1)

	assert.Equal(t, "MISS", w1.Header().Get("X-Cache"))

	// Invalidate cache
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	InvalidateCache(cacheInstance, c)

	// Second request - should be a miss since cache was invalidated
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, "MISS", w2.Header().Get("X-Cache"))
}
