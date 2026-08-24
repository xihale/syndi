package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestZaobaoRealtimeChinaLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(ZaobaoRealtimeHandler, map[string]string{"section": "-"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	first := feed.Items[0]
	if first.Title == "" || first.Link == "" {
		t.Fatalf("expected title and link, got %q %q", first.Title, first.Link)
	}
	if len(first.Description) < 50 {
		t.Fatalf("expected article body description, got %d chars", len(first.Description))
	}
	t.Logf("got %d items, first: %s (%s), desc %d chars", len(feed.Items), first.Title, first.PubDate.Format("2006-01-02 15:04"), len(first.Description))
}

func TestZaobaoRealtimeFinanceLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(ZaobaoRealtimeHandler, map[string]string{"section": "zfinance"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
