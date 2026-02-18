package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rsshub/go/pkg/logger"
	"go.uber.org/zap"
)

const loggerStartTimeKey = "_gin_rsshub_start_time"

// Logger returns a gin middleware that logs HTTP requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store start time
		start := timeNow()
		c.Set(loggerStartTimeKey, start)

		// Path to log
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Build log fields
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.String("duration", formatDuration(duration)),
			zap.String("ip", c.ClientIP()),
		}

		// Add query string if present
		if raw != "" {
			fields = append(fields, zap.String("query", raw))
		}

		// Add user agent if meaningful
		if ua := c.Request.UserAgent(); ua != "" {
			fields = append(fields, zap.String("user-agent", ua))
		}

		// Add error if handler set one
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// Log based on status code
		status := c.Writer.Status()
		switch {
		case status >= 500:
			// Server error
			logger.Error("Request completed with server error", fields...)
		case status >= 400:
			// Client error
			logger.Warn("Request completed with client error", fields...)
		case status >= 300:
			// Redirect
			logger.Debug("Request redirected", fields...)
		default:
			// Success
			logger.Info("Request completed", fields...)
		}
	}
}

// colorForStatus returns ANSI color code for status (console logging)
// This is useful for development mode
func colorForStatus(status int) string {
	switch {
	case status >= 500:
		return "\033[31m" // Red
	case status >= 400:
		return "\033[33m" // Yellow
	case status >= 300:
		return "\033[36m" // Cyan
	case status >= 200:
		return "\033[32m" // Green
	default:
		return "\033[0m" // Reset
	}
}

// resetColor resets ANSI color codes
func resetColor() string {
	return "\033[0m"
}
