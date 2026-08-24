package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestLeetCodeDailyCNLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(LeetCodeDailyQuestionCNHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	item := feed.Items[0]
	if item.PubDate.IsZero() {
		t.Log("warning: daily question has no pubDate")
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), item.Title, item.Link)
}

func TestLeetCodeDailyENLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(LeetCodeDailyQuestionENHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	item := feed.Items[0]
	if item.PubDate.IsZero() {
		t.Log("warning: daily question has no pubDate")
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), item.Title, item.Link)
}
