package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLogger_SuccessRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logger())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
}

func TestLogger_ErrorRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logger())
	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLogger_WithQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logger())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?foo=bar&baz=qux", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_WithUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logger())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_StartTimeRecorded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logger())
	router.GET("/test", func(c *gin.Context) {
		// Check start time was set
		startTime, exists := c.Get(loggerStartTimeKey)
		assert.True(t, exists)
		assert.NotNil(t, startTime)

		if start, ok := startTime.(time.Time); ok {
			// Should be recent
			assert.True(t, time.Since(start) < time.Second)
		}

		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestColorForStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{name: "2xx success", status: 200, want: "\033[32m"},
		{name: "3xx redirect", status: 301, want: "\033[36m"},
		{name: "4xx client error", status: 404, want: "\033[33m"},
		{name: "5xx server error", status: 500, want: "\033[31m"},
		{name: "unknown", status: 100, want: "\033[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorForStatus(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResetColor(t *testing.T) {
	got := resetColor()
	want := "\033[0m"
	assert.Equal(t, want, got)
}
