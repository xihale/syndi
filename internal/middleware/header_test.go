package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHeader_SetsCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Header(5 * time.Minute))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get(headerAccessControlAllowOrigin))
}

func TestHeader_SetsSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Header(5 * time.Minute))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get(headerXContentTypeOpts))
}

func TestHeader_SetsCacheControl(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ttl := 5 * time.Minute
	router := gin.New()
	router.Use(Header(ttl))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "public, max-age=300", w.Header().Get(headerCacheControl))
}

func TestHeader_DisablesCacheWhenTTLIsZero(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Header(0))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "no-cache", w.Header().Get(headerCacheControl))
}

func TestHeader_SetsRSSHubRouteHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Header(5 * time.Minute))
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "/api/test", w.Header().Get(headerXRSSHubRoute))
}

func TestHeader_OptionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Header(5 * time.Minute))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get(headerAccessControlAllowOrigin))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get(headerAccessControlAllowMethods))
	assert.Equal(t, "Content-Type, Authorization", w.Header().Get(headerAccessControlAllowHeaders))
	assert.Equal(t, "86400", w.Header().Get(headerAccessControlMaxAge))
}

func TestHeader_StoresIfNoneMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Header(5 * time.Minute))
	router.GET("/test", func(c *gin.Context) {
		etag := c.GetHeader("If-None-Match")
		c.String(http.StatusOK, etag)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("If-None-Match", "\"test-etag\"")
	router.ServeHTTP(w, req)

	assert.Equal(t, "\"test-etag\"", w.Body.String())
}

func TestSetCacheStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	setCacheStatus(c, "HIT")
	assert.Equal(t, "HIT", c.Writer.Header().Get(headerRSSHubCacheStatus))

	setCacheStatus(c, "MISS")
	assert.Equal(t, "MISS", c.Writer.Header().Get(headerRSSHubCacheStatus))
}

func TestSetCacheHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		ttl            time.Duration
		expectedHeader string
	}{
		{
			name:           "with TTL",
			ttl:            10 * time.Minute,
			expectedHeader: "public, max-age=600",
		},
		{
			name:           "zero TTL",
			ttl:            0,
			expectedHeader: "no-cache",
		},
		{
			name:           "negative TTL",
			ttl:            -1 * time.Minute,
			expectedHeader: "no-cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request, _ = http.NewRequest("GET", "/test", nil)

			setCacheHeaders(c, tt.ttl)
			assert.Equal(t, tt.expectedHeader, c.Writer.Header().Get(headerCacheControl))
		})
	}
}

func TestSetETag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		body          []byte
		ifNoneMatch   string
		expectMatch   bool
		expect304     bool
	}{
		{
			name:        "empty body",
			body:        []byte{},
			ifNoneMatch: "",
			expectMatch: false,
			expect304:   false,
		},
		{
			name:        "no if-none-match header",
			body:        []byte("test content"),
			ifNoneMatch: "",
			expectMatch: false,
			expect304:   false,
		},
		{
			name:        "matching etag",
			body:        []byte("test content"),
			ifNoneMatch: "", // Will be set by generateETagFromBody
			expectMatch: true,
			expect304:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request, _ = http.NewRequest("GET", "/test", nil)

			if tt.ifNoneMatch != "" {
				c.Request.Header.Set("If-None-Match", tt.ifNoneMatch)
			}

			// First, set the ETag from the body to get the expected value
			etag := generateETagFromBody(tt.body)

			// Then test with the If-None-Match header set to that ETag
			if tt.expectMatch {
				c.Request.Header.Set("If-None-Match", etag)
			}

			matches := setETag(c, tt.body)

			assert.Equal(t, tt.expectMatch, matches)
			assert.Equal(t, etag, c.Writer.Header().Get(headerETag))
		})
	}
}
