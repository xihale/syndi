package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

// Live tests hit m.weibo.cn directly. The keyword/super-index search APIs sit
// behind the Sina Visitor System: without WEIBO_COOKIES the upstream answers
// ok=-100, so these tests skip unless both env vars are provided.
func TestWeiboKeywordLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	if os.Getenv("WEIBO_COOKIES") == "" {
		t.Skip("set WEIBO_COOKIES to run the weibo live test")
	}
	feed, err := testutil.RunHandler(WeiboKeywordHandler, map[string]string{"keyword": "RSSHub"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestWeiboSuperIndexLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	if os.Getenv("WEIBO_COOKIES") == "" {
		t.Skip("set WEIBO_COOKIES to run the weibo live test")
	}
	feed, err := testutil.RunHandler(WeiboSuperIndexHandler,
		map[string]string{"id": "1008084989d223732bf6f02f75ea30efad58a9", "type": "sort_time"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestWeiboSearchHotLive(t *testing.T) {
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
