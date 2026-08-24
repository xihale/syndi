package routes

import (
	"os"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/testutil"
)

// Both routes require WEIBO_COOKIES (Sina Visitor System blocks anonymous
// access with HTTP 432), so live tests skip fast when the variable is unset.
func TestWeiboUserLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	if os.Getenv("WEIBO_COOKIES") == "" {
		t.Skip("set WEIBO_COOKIES to run the weibo live test")
	}
	feed, err := testutil.RunHandler(WeiboUserHandler, map[string]string{"uid": "1195230310"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestWeiboHotSearchLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	if os.Getenv("WEIBO_COOKIES") == "" {
		t.Skip("set WEIBO_COOKIES to run the weibo live test")
	}
	feed, err := testutil.RunHandler(WeiboHotSearchHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestParseWeiboDate(t *testing.T) {
	if got := parseWeiboDate("Sun Mar 10 12:00:00 +0800 2024"); got.IsZero() {
		t.Fatal("absolute form should parse")
	}
	if !parseWeiboDate("").IsZero() {
		t.Fatal("empty should be zero")
	}
	if got := parseWeiboDate("3小时前"); !got.After(time.Now().Add(-4 * time.Hour)) {
		t.Fatalf("relative hours should resolve near now, got %v", got)
	}
	if got := parseWeiboDate("5分钟前"); !got.After(time.Now().Add(-10 * time.Minute)) {
		t.Fatalf("relative minutes should resolve near now, got %v", got)
	}
}
