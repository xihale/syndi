package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func odailyLiveGuard(t *testing.T) {
	t.Helper()
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
}

func TestOdailyPostsLive(t *testing.T) {
	odailyLiveGuard(t)
	feed, err := testutil.RunHandler(OdailyPostHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Items[0].Description == "" {
		t.Fatal("expected article content in description")
	}
	t.Logf("posts: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestOdailyNewsflashLive(t *testing.T) {
	odailyLiveGuard(t)
	feed, err := testutil.RunHandler(OdailyNewsflashHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Items[0].GUID == "" {
		t.Fatal("expected GUID set")
	}
	t.Logf("newsflash: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestOdailyHotWeeklyLive(t *testing.T) {
	odailyLiveGuard(t)
	feed, err := testutil.RunHandler(OdailyHotHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("hot weekly: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestOdailyHotDailyLive(t *testing.T) {
	odailyLiveGuard(t)
	feed, err := testutil.RunHandler(OdailyHotHandler, map[string]string{"period": "daily"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("hot daily: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
