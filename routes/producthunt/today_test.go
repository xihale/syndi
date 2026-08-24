package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestTodayLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(TodayHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	first := feed.Items[0]
	if first.Title == "" || first.Link == "" {
		t.Fatal("expected item title/link")
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), first.Title, first.Link)
}
