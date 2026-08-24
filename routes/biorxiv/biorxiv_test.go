package routes

import (
	"os"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestBiorxivLatestLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(BiorxivLatestHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) < 30 {
		t.Fatalf("expected at least 30 items, got %d", len(feed.Items))
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), feed.Items[0].Title, feed.Items[0].Link)
}

func TestMedrxivLatestLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(MedrxivLatestHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), feed.Items[0].Title, feed.Items[0].Link)
}
