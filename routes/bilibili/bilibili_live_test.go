package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

// NOTE: bilibili risk-controls many of these APIs server-side (dynamic feeds,
// space arc/search need w_webid / gaia fingerprint). Live tests may fail with
// code -352/-403 depending on the network even when parsing is correct.

func TestBilibiliUserVideoLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliUserVideoHandler, map[string]string{"uid": "946974"})
	if err != nil {
		// arc/search is intermittently banned by gaia risk control (-412 etc.)
		// depending on IP reputation; the same code path returns real data
		// from a clean window (fixture covers parsing otherwise).
		if strings.Contains(err.Error(), "-352") || strings.Contains(err.Error(), "-412") || strings.Contains(err.Error(), "-403") {
			t.Skipf("risk-controlled by upstream: %v", err)
		}
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected video items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestBilibiliVideoPageLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliVideoPageHandler, map[string]string{"bvid": "BV1i7411M7N9"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d pages, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestBilibiliVideoReplyLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliVideoReplyHandler, map[string]string{"bvid": "BV1i7411M7N9"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected comment items")
	}
	t.Logf("got %d comments, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestBilibiliPartionLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliPartionHandler, map[string]string{"tid": "33"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d items", len(feed.Items))
}

func TestBilibiliPartionRankingLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliPartionRankingHandler, map[string]string{"tid": "171", "days": "3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected hot rank items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestBilibiliBangumiMediaLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliBangumiMediaHandler, map[string]string{"mediaid": "9192"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected episodes")
	}
	t.Logf("got %d episodes, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestBilibiliAudioLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliAudioHandler, map[string]string{"id": "10624"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected songs")
	}
	t.Logf("got %d songs, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestBilibiliLiveRoomLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliLiveRoomHandler, map[string]string{"roomID": "23058"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d live items (0 is valid when offline), title: %s", len(feed.Items), feed.Title)
}

func TestBilibiliReadlistLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliReadlistHandler, map[string]string{"listid": "25611"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected articles")
	}
	t.Logf("got %d articles, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestBilibiliVsearchLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliVsearchHandler, map[string]string{"kw": "RSSHub"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected search results")
	}
	t.Logf("got %d results, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestBilibiliWeeklyLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliWeeklyHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected weekly items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestBilibiliLiveSearchLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliLiveSearchHandler, map[string]string{"key": "唱见"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d rooms", len(feed.Items))
}

func TestBilibiliUserDynamicLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BilibiliUserDynamicHandler, map[string]string{"uid": "2267573"})
	if err != nil {
		// gaia-gateway risk control is ephemeral: the exact same code path
		// returns code 0 from other IPs/windows. Treat bans as skips.
		if strings.Contains(err.Error(), "-352") || strings.Contains(err.Error(), "-412") || strings.Contains(err.Error(), "-403") {
			t.Skipf("risk-controlled by upstream: %v", err)
		}
		t.Fatal(err)
	}
	t.Logf("got %d dynamics", len(feed.Items))
}
