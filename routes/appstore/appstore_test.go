package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestAppstoreXianmianLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(AppstoreXianmianHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestAppstorePriceLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(AppstorePriceHandler, map[string]string{
		"country": "us", "type": "ios", "id": "id1444383602",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d items (0 = no active price drop), title: %s", len(feed.Items), feed.Title)
}
