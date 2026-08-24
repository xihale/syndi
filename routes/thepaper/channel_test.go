package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestThePaperChannelLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(ThePaperChannelHandler, map[string]string{"id": "25950"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Items[0].Title == "" || feed.Items[0].Link == "" {
		t.Fatal("expected first item to have title and link")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
