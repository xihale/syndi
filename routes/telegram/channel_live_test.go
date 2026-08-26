package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestTelegramChannelLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(TelegramChannelHandler, map[string]string{"username": "durov"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if len(feed.Items) > 1 && feed.Items[0].PubDate.Before(feed.Items[len(feed.Items)-1].PubDate) {
		t.Fatal("expected newest-first item ordering")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestTelegramChannelRouteParamsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(TelegramChannelHandler, map[string]string{
		"username":    "durov",
		"routeParams": "/showLinkPreview=0&showMessageMedia=0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	for _, item := range feed.Items {
		if item.Description == "" {
			t.Fatalf("expected description for %q", item.GUID)
		}
	}
	t.Logf("got %d items with stripped switches", len(feed.Items))
}
