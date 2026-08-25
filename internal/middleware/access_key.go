package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

const accessKeyHeader = "X-Access-Key"

// AccessKey returns a middleware enforcing the configured access key.
//
// When key is empty the instance stays open (default). Otherwise every
// request must present the key via the "key" query parameter or the
// X-Access-Key header; /status remains open for health checks.
func AccessKey(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key == "" || c.Request.URL.Path == "/status" {
			c.Next()
			return
		}

		presented := c.Query("key")
		if presented == "" {
			presented = c.GetHeader(accessKeyHeader)
		}
		if subtle.ConstantTimeCompare([]byte(presented), []byte(key)) == 1 {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or missing access key",
		})
	}
}
