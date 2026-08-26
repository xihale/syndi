package routes

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/testutil"
)

func TestV2EXHotLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(V2EXHotHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestV2EXLatestLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(V2EXLatestHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestV2EXNodeLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(V2EXNodeHandler, map[string]string{"name": "python"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestV2EXTopicLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(V2EXTopicHandler, map[string]string{"id": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestV2EXPostLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(V2EXPostHandler, map[string]string{"postid": "584403"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range feed.Items {
		if !strings.HasPrefix(item.GUID, "v2ex-reply-") {
			t.Fatalf("unexpected GUID %q", item.GUID)
		}
	}
	t.Logf("got %d replies", len(feed.Items))
}

func TestV2EXTopicsTypeLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	for _, typ := range []string{"hot", "latest"} {
		feed, err := testutil.RunHandler(V2EXTopicsHandler, map[string]string{"type": typ})
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		if len(feed.Items) == 0 {
			t.Fatalf("%s: expected topics", typ)
		}
		t.Logf("%s: got %d topics, first: %s", typ, len(feed.Items), feed.Items[0].Title)
	}

	if _, err := testutil.RunHandler(V2EXTopicsHandler, map[string]string{"type": "everything"}); err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestV2EXTabLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(V2EXTabHandler, map[string]string{"tabid": "daily"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected tab topics")
	}
	first := feed.Items[0]
	if first.Link == "" || !strings.Contains(first.GUID, "v2ex-topic-") {
		t.Fatalf("unexpected first tab item: %+v", first)
	}
	t.Logf("got %d topics, first: %s (%s)", len(feed.Items), first.Title, first.PubDate.Format(time.RFC3339))
}
