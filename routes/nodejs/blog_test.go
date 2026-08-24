package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestNodejsBlogLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NodejsBlogHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Title == "" || feed.Link == "" {
		t.Fatal("expected normalized feed title/link")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
