package routes

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/xihale/syndi/internal/testutil"
)

func TestJikeTopicTextLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(JikeTopicTextHandler, map[string]string{"id": "553870e8e4b0cafb0a1bef68"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Items[0].PubDate.IsZero() {
		t.Log("warning: first post has no pubDate")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestJikeTopicLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(JikeTopicHandler, map[string]string{"id": "553870e8e4b0cafb0a1bef68"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestJikeUserLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(JikeUserHandler, map[string]string{"id": "3EE02BC9-C5B3-4209-8750-4ED1EE0F67BB"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestJikeRouteRegistration(t *testing.T) {
	engine := gin.New()
	for _, spec := range Routes {
		path := spec.Path
		engine.GET("/rss/jike/"+path, func(c *gin.Context) {})
	}
	registered := map[string]bool{}
	for _, r := range engine.Routes() {
		registered[r.Path] = true
	}
	for _, want := range []string{"/rss/jike/topic/:id", "/rss/jike/topic/text/:id", "/rss/jike/user/:id"} {
		if !registered[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}
