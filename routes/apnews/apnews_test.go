package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/testutil"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

func TestAPNewsLatestLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(APNewsHandler, nil)
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
	t.Logf("got %d items, first: %s (%s)", len(feed.Items), first.Title, first.PubDate.Format("2006-01-02 15:04"))
}

func TestAPNewsFullTextLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(APNewsHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) < 3 {
		t.Fatalf("expected at least 3 items, got %d", len(feed.Items))
	}
	req := httptest.NewRequest(http.MethodGet, "/live-test", nil)
	c := ctxpkg.NewContext(httptest.NewRecorder(), req)
	cl := client.New(client.WithTimeout(30 * time.Second))
	c.SetClient(cl)

	var item *models.Item
	for i := range feed.Items {
		if len(feed.Items[i].Categories) > 0 && feed.Items[i].Categories[0] == "eng" {
			item = &feed.Items[i]
			break
		}
	}
	if item == nil {
		item = &feed.Items[0]
	}
	apEnrichWithFullText(c.Parent(), cl, item)
	if item.Description == "" {
		t.Fatalf("expected enriched description for %s", item.Link)
	}
	t.Logf("enriched %q -> desc %d chars", item.Title, len(item.Description))
}
