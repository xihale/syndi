package routes

import (
	"os"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestSteamNewsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(SteamNewsHandler, map[string]string{"appid": "570"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestSteamSpecialsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(SteamSpecialsHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	item := feed.Items[0]
	if item.Link == "" || item.Title == "" {
		t.Fatalf("item missing title/link: %+v", item)
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), item.Title, item.Link)
}
