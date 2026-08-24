package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// MiddlewareError represents an error with additional context
type MiddlewareError struct {
	Err     error
	Message string
	Code    int
}

// Error implements the error interface
func (e *MiddlewareError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *MiddlewareError) Unwrap() error {
	return e.Err
}

// generateETagFromBody generates an ETag from response body
func generateETagFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	hash := sha256.Sum256(body)
	return `"` + hex.EncodeToString(hash[:]) + `"`
}

// timeNow returns the current time in UTC
func timeNow() time.Time {
	return time.Now().UTC()
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%d ns", d.Nanoseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.2f ms", float64(d.Nanoseconds())/1000000.0)
	}
	return fmt.Sprintf("%.2f s", d.Seconds())
}
