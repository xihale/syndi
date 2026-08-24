package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestMastodonAccountLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	params := map[string]string{"instance": "mastodon.social", "id": "1"}
	feed, err := testutil.RunHandler(MastodonAccountHandler, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestMastodonTimelineLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	// mastodon.social now requires auth for its public timeline; use an
	// instance that still exposes it anonymously.
	params := map[string]string{"instance": "fosstodon.org"}
	feed, err := testutil.RunHandler(MastodonTimelineHandler, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
