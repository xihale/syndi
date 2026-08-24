package routes

import (
	"os"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestAMOAddonLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(AMOAddonHandler, map[string]string{"slug": "ublock-origin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Items[0].PubDate.IsZero() {
		t.Fatal("expected file.created date on first item")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
