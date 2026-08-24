package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestPubmedSearchLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(PubmedSearchHandler, map[string]string{"term": "crispr"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) < 10 {
		t.Fatalf("expected at least 10 items, got %d", len(feed.Items))
	}
	item := feed.Items[0]
	if item.Link == "" || item.PubDate.IsZero() {
		t.Fatalf("expected link and pubdate on first item: %+v", item)
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), item.Title, item.Link)
}
