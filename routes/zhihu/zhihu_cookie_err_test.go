package routes

import (
	"strings"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestZhihuCookieRequiredFailsFast(t *testing.T) {
	if zhihuCookies() != "" {
		t.Skip("cookie configured; fail-fast path not exercised")
	}
	_, err := testutil.RunHandler(ZhihuZhuanlanHandler, map[string]string{"id": "googledevelopers"})
	if err == nil {
		t.Fatal("expected error without cookie")
	}
	if !strings.Contains(err.Error(), "ZHIHU_COOKIES") {
		t.Fatalf("error should mention env var, got: %v", err)
	}
	t.Logf("fail-fast error: %v", err)
}
