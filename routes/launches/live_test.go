package routes

import (
	"os"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestLaunchesUpcomingLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(LaunchesUpcomingHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if len(feed.Items) > 10 {
		t.Fatalf("expected at most 10 items, got %d", len(feed.Items))
	}
	t.Logf("got %d items, first: %s (net %s)", len(feed.Items), feed.Items[0].Title, feed.Items[0].PubDate)
}
