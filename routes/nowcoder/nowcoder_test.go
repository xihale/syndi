package routes

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/xihale/syndi/internal/testutil"
)

func TestNowCoderHotsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NowCoderHotsHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestNowCoderTopHotLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NowCoderHotsHandler, map[string]string{"type": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestNowCoderScheduleLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NowCoderScheduleHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestNowCoderInterviewLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NowCoderInterviewHandler, map[string]string{"jobId": "11200"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Items[0].PubDate.IsZero() {
		t.Log("warning: first interview item has no pubDate")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestNowCoderRouteRegistration(t *testing.T) {
	engine := gin.New()
	for _, spec := range Routes {
		path := spec.Path
		engine.GET("/rss/nowcoder/"+path, func(c *gin.Context) {})
	}
	registered := map[string]bool{}
	for _, r := range engine.Routes() {
		registered[r.Path] = true
	}
	for _, want := range []string{
		"/rss/nowcoder/hots",
		"/rss/nowcoder/hots/:type",
		"/rss/nowcoder/schedule",
		"/rss/nowcoder/schedule/:propertyId",
		"/rss/nowcoder/schedule/:propertyId/:typeId",
		"/rss/nowcoder/interview/:jobId",
	} {
		if !registered[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}
