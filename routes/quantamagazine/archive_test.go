package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestArchiveLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(ArchiveHandler, nil)
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
	if len(first.Description) < 500 {
		t.Fatalf("expected substantial article content, got %d chars", len(first.Description))
	}
	if first.PubDate.IsZero() {
		t.Log("warning: first item has zero pubDate")
	}
	t.Logf("got %d items, first: %s (%s), desc %d chars",
		len(feed.Items), first.Title, first.PubDate.Format("2006-01-02"), len(first.Description))
}
