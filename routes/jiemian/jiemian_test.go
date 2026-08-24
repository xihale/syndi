package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestJiemianHomeLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(JiemianHomeHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	first := feed.Items[0]
	if first.Title == "" || first.Link == "" {
		t.Fatal("expected title and link on first item")
	}
	t.Logf("got %d items, first: %s | %s", len(feed.Items), first.Title, first.Link)
}

func TestJiemianListsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(JiemianListsHandler, map[string]string{"id": "4"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	first := feed.Items[0]
	if first.PubDate.IsZero() {
		t.Log("note: pubDate missing on first item")
	}
	if len(first.Description) < 100 {
		t.Fatalf("expected article content in description, got: %.200s", first.Description)
	}
	t.Logf("got %d items, first: %s (%d chars desc)", len(feed.Items), first.Title, len(first.Description))
}
