package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func pixivLiveGuard(t *testing.T) {
	t.Helper()
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	if os.Getenv(pixivCookiesEnv) == "" {
		t.Skipf("%s not set; pixiv routes need a logged-in cookie", pixivCookiesEnv)
	}
}

func TestPixivCookiesRequiredFailsFast(t *testing.T) {
	if pixivCookies() != "" {
		t.Skip("cookie configured; fail-fast path not exercised")
	}
	_, err := testutil.RunHandler(PixivSearchHandler, map[string]string{"keyword": "GenshinImpact"})
	if err == nil {
		t.Fatal("expected error without cookie")
	}
	if !strings.Contains(err.Error(), pixivCookiesEnv) {
		t.Fatalf("error should mention env var, got: %v", err)
	}
	t.Logf("fail-fast error: %v", err)
}

func TestPixivUserLive(t *testing.T) {
	pixivLiveGuard(t)
	feed, err := testutil.RunHandler(PixivUserHandler, map[string]string{"id": "15288095"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Items[0].Description == "" {
		t.Fatal("expected caption/images in description")
	}
	t.Logf("user: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestPixivSearchLive(t *testing.T) {
	pixivLiveGuard(t)
	feed, err := testutil.RunHandler(PixivSearchHandler, map[string]string{"keyword": "GenshinImpact"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("search: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestPixivNovelSearchLive(t *testing.T) {
	pixivLiveGuard(t)
	feed, err := testutil.RunHandler(PixivNovelSearchHandler, map[string]string{"keyword": "原神"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("novel search: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestOrderedProfileIllustIDs(t *testing.T) {
	raw := []byte(`{"error":false,"message":"","body":{"illusts":{"111":null,"999":null,"42":{"id":"42"}},"manga":[],"novels":[],"pickup":[]}}`)
	ids, err := orderedProfileIllustIDs(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] != "111" || ids[2] != "42" {
		t.Fatalf("unexpected ordered ids: %v", ids)
	}
	missing, err := orderedProfileIllustIDs([]byte(`{"error":false,"body":{"illusts":{},"manga":[]}}`))
	if err != nil || missing != nil {
		t.Fatalf("expected empty result, got %v / %v", missing, err)
	}
}
