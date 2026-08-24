package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestHackageRecentLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(HackageRecentHandler, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Items[0].PubDate.IsZero() {
		t.Fatal("expected upload time on first item")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
