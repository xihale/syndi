package routes

import (
	"os"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestUsgsSignificantLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(USGSSignificantHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Some days legitimately have zero significant quakes; only log.
	first := ""
	if len(feed.Items) > 0 {
		first = feed.Items[0].Title
	}
	t.Logf("got %d items, first: %q", len(feed.Items), first)
}

func TestUsgsAllDayLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(USGSAllDayHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if len(feed.Items) > usgsMaxItems {
		t.Fatalf("expected at most %d items, got %d", usgsMaxItems, len(feed.Items))
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
