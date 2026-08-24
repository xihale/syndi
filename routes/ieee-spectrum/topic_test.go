package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestIEEESpectrumTopicLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(IEEESpectrumTopicHandler, map[string]string{"topic": "artificial-intelligence"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestIEEESpectrumTopicRejectsBadSlug(t *testing.T) {
	if _, err := testutil.RunHandler(IEEESpectrumTopicHandler, map[string]string{"topic": "AI Robots"}); err == nil {
		t.Fatal("expected error for invalid slug")
	}
}
