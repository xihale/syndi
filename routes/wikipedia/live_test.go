package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestWikipediaOnThisDayLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(WikipediaOnThisDayHandler, map[string]string{"monthday": "08-24"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	item := feed.Items[0]
	if item.Link == "" {
		t.Fatal("first item missing link")
	}
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), item.Title, item.PubDate)
}

func TestWikipediaFeaturedLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(WikipediaFeaturedHandler, map[string]string{"date": "2026-08-23"})
	if err != nil {
		t.Fatal(err)
	}
	// Expect TFA + top-3 most-read (+ optional picture of the day).
	if len(feed.Items) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(feed.Items))
	}
	for _, item := range feed.Items {
		if !strings.HasPrefix(item.GUID, "wikipedia-") {
			t.Fatalf("unexpected GUID %q", item.GUID)
		}
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestValidateMonthDay(t *testing.T) {
	if _, _, err := validateMonthDay("08-24"); err != nil {
		t.Fatalf("valid monthday rejected: %v", err)
	}
	for _, bad := range []string{"8-24", "0824", "13-01", "00-10", "08-32", "ab-cd", ""} {
		if _, _, err := validateMonthDay(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestTruncateTitle(t *testing.T) {
	short := truncateTitle("Already short", 120)
	if short != "Already short" {
		t.Fatalf("short title altered: %q", short)
	}
	long := truncateTitle(strings.Repeat("word ", 60), 120)
	if len(long) > 124 || !strings.HasSuffix(long, "…") {
		t.Fatalf("long title not truncated: %q (len %d)", long, len(long))
	}
}
