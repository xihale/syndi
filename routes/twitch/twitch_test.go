package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestTwitchLiveLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(TwitchLiveHandler, map[string]string{"login": "riotgames"})
	if err != nil {
		t.Fatal(err)
	}
	if feed.Title == "" || feed.Link == "" {
		t.Fatal("expected normalized feed title/link")
	}
	// Channel may be offline; the route is allowed to return zero items.
	t.Logf("got %d items (0 = channel offline), title: %s", len(feed.Items), feed.Title)
}

func TestTwitchScheduleLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(TwitchScheduleHandler, map[string]string{"login": "northernlion"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Log("channel has no published schedule segments right now")
		return
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestTwitchVideoLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	for _, filter := range []string{"all", "archive", "highlights"} {
		feed, err := testutil.RunHandler(TwitchVideoHandler, map[string]string{"login": "riotgames", "filter": filter})
		if err != nil {
			t.Fatalf("%s: %v", filter, err)
		}
		if len(feed.Items) == 0 {
			t.Fatalf("%s: expected items", filter)
		}
		t.Logf("%s: got %d items, first: %s", filter, len(feed.Items), feed.Items[0].Title)
	}
}
