package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rsshub/go/pkg/logger"
	"go.uber.org/zap"
)

const recoveryLogKey = "_gin_rsshub_recovery"

// Recovery returns a gin middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := debug.Stack()

				// Log the panic with stack trace
				logger.Error("Request panic recovered",
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.String("ip", c.ClientIP()),
					zap.String("user-agent", c.Request.UserAgent()),
					zap.ByteString("stack", stack),
				)

				// Store panic in context for potential use by other middleware
				c.Set(recoveryLogKey, err)

				// Return 500 Internal Server Error
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal Server Error",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// GetPanicError retrieves the panic error from the Gin context if one occurred
func GetPanicError(c *gin.Context) (interface{}, bool) {
	if val, exists := c.Get(recoveryLogKey); exists {
		return val, true
	}
	return nil, false
}

// formatPanicError formats a panic error into a string
func formatPanicError(err interface{}) string {
	switch err := err.(type) {
	case error:
		return err.Error()
	case string:
		return err
	default:
		return fmt.Sprintf("%v", err)
	}
}
