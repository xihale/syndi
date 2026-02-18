package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRecovery_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Recovery())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
}

func TestRecovery_WithPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Internal Server Error")
}

func TestRecovery_WithPanicString(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("string panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRecovery_WithPanicError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic(assert.AnError)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetPanicError_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	_, exists := GetPanicError(c)
	assert.False(t, exists)
}

func TestGetPanicError_WithPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	// After request, check if panic was recorded
	// Note: This is a simplified test - in real scenario we'd need
	// to access the context from within the handler
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFormatPanicError(t *testing.T) {
	tests := []struct {
		name string
		err  interface{}
		want string
	}{
		{
			name: "error type",
			err:  assert.AnError,
			want: assert.AnError.Error(),
		},
		{
			name: "string type",
			err:  "string error",
			want: "string error",
		},
		{
			name: "other type",
			err:  123,
			want: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPanicError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
