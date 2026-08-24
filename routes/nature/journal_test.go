package routes

import (
	"os"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestNatureJournalLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NatureJournalHandler, map[string]string{"journal": "nature"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestNatureJournalRejectsBadSlug(t *testing.T) {
	if _, err := testutil.RunHandler(NatureJournalHandler, map[string]string{"journal": "Nature2"}); err == nil {
		t.Fatal("expected error for invalid slug")
	}
}
