package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestSpaceflightNewsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(SpaceflightNewsHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if len(feed.Items) > 20 {
		t.Fatalf("expected at most 20 items, got %d", len(feed.Items))
	}
	item := feed.Items[0]
	if !strings.HasPrefix(item.Link, "https://") {
		t.Fatalf("unexpected first link: %q", item.Link)
	}
	t.Logf("got %d items, first: %s", len(feed.Items), item.Title)
}
