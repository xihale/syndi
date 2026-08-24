package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestSmashingLatestLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(LatestHandler, nil)
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

func TestSmashingCategoryLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(CategoryHandler, map[string]string{"category": "css"})
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
	if first.PubDate.IsZero() {
		t.Log("warning: first item has zero pubDate")
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), first.Title, first.PubDate.Format("2006-01-02"))
}
