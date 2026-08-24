package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestVSCodeExtensionLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(VSCodeExtensionHandler, map[string]string{"publisher": "esbenp", "name": "prettier-vscode"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	for _, item := range feed.Items {
		if item.PubDate.IsZero() {
			t.Fatalf("expected lastUpdated date on %s", item.Title)
		}
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), feed.Items[0].Title, feed.Items[0].PubDate)
}
