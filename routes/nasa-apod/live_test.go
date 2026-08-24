package routes

import (
	"os"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestNasaApodLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NASAAPODHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	for _, item := range feed.Items {
		if item.Link == "" || item.GUID == "" {
			t.Fatalf("item missing link/guid: %+v", item.Title)
		}
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), feed.Items[0].Title, feed.Items[0].Link)
}

func TestApodCompactDate(t *testing.T) {
	tests := map[string]string{
		"2008-04-20": "080420",
		"2026-01-01": "260101",
		"bad":        "",
	}
	for in, want := range tests {
		if got := apodCompactDate(in); got != want {
			t.Fatalf("apodCompactDate(%q) = %q, want %q", in, got, want)
		}
	}
}
