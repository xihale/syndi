package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestCNBCRSSLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(CNBCRSSHandler, map[string]string{"id": "-"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	first := feed.Items[0]
	t.Logf("got %d items, first: %s (desc %d chars)", len(feed.Items), first.Title, len(first.Description))
	if len(first.Description) < 200 || strings.Contains(first.Description, "<style") {
		t.Fatalf("expected enriched full-text description, got %d chars", len(first.Description))
	}
}
