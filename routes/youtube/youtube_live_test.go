package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestYouTubeChannelLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(YouTubeChannelHandler, map[string]string{"id": "UCX6OQ3DkcsbYNE6H8uQQuVA"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s | %s", len(feed.Items), feed.Items[0].Title, feed.Items[0].Link)
}

func TestYouTubeChannelShortsFilterLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(YouTubeChannelHandler, map[string]string{
		"id":          "UCXuqSBlHAE6Xw-yeJA0Tunw",
		"routeParams": "/embed=0&filterShorts=0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	for _, item := range feed.Items {
		if strings.Contains(item.Description, "<iframe") {
			t.Fatalf("unexpected iframe with embed=0 in %q", item.GUID)
		}
	}
	t.Logf("got %d embed-less items", len(feed.Items))
}

func TestYouTubeUserLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(YouTubeUserHandler, map[string]string{"username": "@JFlaMusic"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestYouTubeLegacyUserLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(YouTubeUserHandler, map[string]string{"username": "Google"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items via ?user= feed", len(feed.Items))
}

func TestYouTubeCustomURLLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(YouTubeCustomHandler, map[string]string{"username": "TED"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items from /c/TED, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestYouTubePlaylistLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(YouTubePlaylistHandler, map[string]string{"id": "PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
