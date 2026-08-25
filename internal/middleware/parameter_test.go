package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xihale/syndi/pkg/models"
)

// Regression test: cached feeds are shared instances (see pkg/cache
// MemoryCache). Query-parameter processing must never mutate the caller's
// backing array — a single ?sorted=false&brief=N request used to truncate
// the cached entry for every subsequent reader.
func TestProcessFeedDoesNotMutateInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	original := "0123456789abcdefghij"
	items := []models.Item{{Title: "a", Description: original}}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/x?sorted=false&brief=5", nil)

	got := ProcessFeed(c, items)

	if items[0].Description != original {
		t.Fatalf("ProcessFeed mutated the caller's (possibly cached) items: %q", items[0].Description)
	}
	if got[0].Description != "01234..." {
		t.Errorf("brief truncation = %q, want %q", got[0].Description, "01234...")
	}
}

// Truncation must respect rune boundaries for multi-byte text.
func TestProcessFeedBriefMultibyte(t *testing.T) {
	gin.SetMode(gin.TestMode)

	items := []models.Item{{Title: "a", Description: "你好世界测试文本"}}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/x?brief=3", nil)

	got := ProcessFeed(c, items)

	if got[0].Description != "你好世..." {
		t.Errorf("brief multibyte = %q, want %q", got[0].Description, "你好世...")
	}
}
