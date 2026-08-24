package routes

import (
	"os"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestDockerHubTagsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(DockerHubTagsHandler, map[string]string{"repo": "library/nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	for _, item := range feed.Items {
		if item.PubDate.IsZero() {
			t.Fatalf("expected tag_last_pushed date on %s", item.Title)
		}
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
