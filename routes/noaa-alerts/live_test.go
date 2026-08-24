package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestNoaaAlertsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NOAAAlertsHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if len(feed.Items) > noaaMaxAlerts {
		t.Fatalf("expected at most %d items, got %d", noaaMaxAlerts, len(feed.Items))
	}
	item := feed.Items[0]
	if item.Link == "" || !strings.HasPrefix(item.Link, "https://api.weather.gov/alerts/") {
		t.Fatalf("unexpected first link: %q", item.Link)
	}
	t.Logf("got %d items, first: %s", len(feed.Items), item.Title)
}
