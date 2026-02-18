package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xihale/rsshub-go/pkg/models"
)

// responseWriterWrapper wraps gin.ResponseWriter to capture response body
type responseWriterWrapper struct {
	gin.ResponseWriter
	body        *bytes.Buffer
	wroteHeader bool
	statusCode  int
	buffered    bool // If true, don't write to underlying writer
}

// newResponseWriterWrapper creates a new response writer wrapper
func newResponseWriterWrapper(w gin.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK, // Default status code
		buffered:       true,          // Buffer by default for middleware
	}
}

// newResponseWriterWrapperUnbuffered creates a wrapper that writes through
func newResponseWriterWrapperUnbuffered(w gin.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
		buffered:       false, // Write through
	}
}

// Write captures the response body
func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if w.body != nil {
		w.body.Write(b)
	}
	// Only write to underlying writer if not buffered
	if !w.buffered {
		return w.ResponseWriter.Write(b)
	}
	return len(b), nil
}

// WriteHeader captures the status code
func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.statusCode = statusCode
		w.wroteHeader = true
		// Only write header to underlying writer if not buffered
		if !w.buffered {
			w.ResponseWriter.WriteHeader(statusCode)
		}
	}
}

// Flush writes the buffered response to the underlying writer
func (w *responseWriterWrapper) Flush() {
	// Write status code if set
	if w.wroteHeader && w.buffered {
		w.ResponseWriter.WriteHeader(w.statusCode)
	}
	// Write buffered body
	if w.body != nil && w.body.Len() > 0 {
		w.ResponseWriter.Write(w.body.Bytes())
	}
}

// Header returns the underlying response writer's header map
// This ensures headers set on the wrapper are visible in the final response
func (w *responseWriterWrapper) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Status returns the captured status code
func (w *responseWriterWrapper) Status() int {
	return w.statusCode
}

// Body returns the captured response body
func (w *responseWriterWrapper) Body() *bytes.Buffer {
	return w.body
}

// Bytes returns the captured response body as bytes
func (w *responseWriterWrapper) Bytes() []byte {
	if w.body != nil {
		return w.body.Bytes()
	}
	return nil
}

// String returns the captured response body as string
func (w *responseWriterWrapper) String() string {
	if w.body != nil {
		return w.body.String()
	}
	return ""
}

// reset resets the wrapper for reuse
func (w *responseWriterWrapper) reset() {
	w.body.Reset()
	w.wroteHeader = false
	w.statusCode = http.StatusOK
}

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

// generateETag generates an ETag from a feed
// The ETag is a SHA256 hash of the feed's JSON representation
func generateETag(feed *models.Feed) string {
	if feed == nil {
		return ""
	}

	// Create a canonical representation for hashing
	data, err := json.Marshal(feed)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return `"` + hex.EncodeToString(hash[:]) + `"`
}

// generateETagFromBody generates an ETag from response body
func generateETagFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	hash := sha256.Sum256(body)
	return `"` + hex.EncodeToString(hash[:]) + `"`
}

// copyReader copies from an io.Reader to a bytes.Buffer
func copyReader(r io.Reader) ([]byte, error) {
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, r)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
