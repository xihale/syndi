package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/client"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// serveTelegramFixture serves the recorded t.me channel preview HTML so the
// handler can be exercised without network access.
func serveTelegramFixture(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "testdata/channel.html")
	}))
	t.Cleanup(server.Close)
	return server
}

// runTelegramChannelFeed invokes the handler against the fixture server.
func runTelegramChannelFeed(t *testing.T, username, routeParams string) (*models.Feed, error) {
	t.Helper()
	server := serveTelegramFixture(t)
	prev := telegramChannelDataURL
	telegramChannelDataURL = server.URL
	t.Cleanup(func() { telegramChannelDataURL = prev })

	params := map[string]string{"username": username}
	if routeParams != "" {
		params["routeParams"] = routeParams
	}
	req := httptest.NewRequest(http.MethodGet, "/rss-test", nil)
	c := ctxpkg.NewContext(httptest.NewRecorder(), req)
	c.SetParams(params)
	c.SetClient(client.New(client.WithTimeout(10 * time.Second)))
	return TelegramChannelHandler(c)
}

func TestTelegramChannelFixture(t *testing.T) {
	feed, err := runTelegramChannelFeed(t, "durov", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(feed.Title, "Pavel Durov") {
		t.Fatalf("unexpected feed title %q", feed.Title)
	}
	if len(feed.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(feed.Items))
	}
	for _, item := range feed.Items {
		if !strings.HasPrefix(item.GUID, "telegram-channel-durov/") {
			t.Fatalf("expected telegram-channel- GUID prefix, got %q", item.GUID)
		}
		if item.PubDate.IsZero() {
			t.Fatalf("expected parsed pubDate for %q", item.GUID)
		}
		if !strings.HasPrefix(item.Link, "https://t.me/durov/") {
			t.Fatalf("unexpected link %q", item.Link)
		}
	}
	// t.me serves the last messages oldest-first in the DOM; the feed mirrors
	// upstream's reversal so items[0] is the newest post.
	first, last := feed.Items[0], feed.Items[len(feed.Items)-1]
	if first.PubDate.Before(last.PubDate) {
		t.Fatalf("expected newest-first ordering, got first=%v last=%v", first.PubDate, last.PubDate)
	}

	// The photo post keeps its caption under a media tag prefix.
	var videoItem, photoItem *models.Item
	for i := range feed.Items {
		if strings.HasPrefix(feed.Items[i].Title, "[Video]") {
			videoItem = &feed.Items[i]
		}
		if strings.HasSuffix(feed.Items[i].GUID, "/536") {
			photoItem = &feed.Items[i]
		}
	}
	if photoItem == nil || !strings.HasPrefix(photoItem.Title, "[Photo]") {
		t.Fatalf("expected [Photo]-tagged item, got %+v", photoItem)
	}
	if !strings.Contains(photoItem.Description, `<img src="https://cdn4.telesco.pe/file/`) {
		t.Fatalf("expected embedded photo image, got %.200q", photoItem.Description)
	}
	if videoItem == nil || !strings.HasPrefix(videoItem.Title, "[Video]") {
		t.Fatalf("expected [Video]-tagged poster item")
	}
}

func TestTelegramChannelFixtureSwitches(t *testing.T) {
	// Disabling media must drop message media anchors (link-preview images
	// inside blockquotes are controlled by showLinkPreview instead).
	feed, err := runTelegramChannelFeed(t, "durov", "/showMessageMedia=0")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range feed.Items {
		if strings.Contains(item.Description, `</a><a href=`) || strings.Contains(item.Description, `"><img`) {
			t.Fatalf("unexpected media markup with showMessageMedia=0 in %q", item.GUID)
		}
		if strings.Contains(item.Description, "telesco.pe") && !strings.Contains(item.Description, "<blockquote>") {
			t.Fatalf("unexpected photo image with showMessageMedia=0 in %q", item.GUID)
		}
		if strings.HasPrefix(item.Title, "[") {
			t.Fatalf("unexpected media tag in title %q", item.Title)
		}
	}

	// Link previews are blockquote cards from the fixture; disabled keeps none.
	withPreview, err := runTelegramChannelFeed(t, "durov", "")
	if err != nil {
		t.Fatal(err)
	}
	hasPreviewInDefault := false
	for _, item := range withPreview.Items {
		if strings.Contains(item.Description, "<blockquote>") {
			hasPreviewInDefault = true
		}
	}
	withoutPreview, err := runTelegramChannelFeed(t, "durov", "/showLinkPreview=0")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range withoutPreview.Items {
		if strings.Contains(item.Description, "<blockquote>") {
			t.Fatalf("unexpected link-preview markup with showLinkPreview=0 in %q", item.GUID)
		}
	}
	if !hasPreviewInDefault {
		t.Log("fixture carries no link-preview card; switch verification is one-sided")
	}
}

func TestTelegramChannelFixtureSearchFallback(t *testing.T) {
	// Unknown tails are treated as search queries (upstream compat).
	feed, err := runTelegramChannelFeed(t, "durov", "/rsshub")
	if err != nil {
		t.Fatalf("search fallback should still parse: %v", err)
	}
	if !strings.Contains(feed.Title, `"rsshub"`) {
		t.Fatalf("search query should appear in feed title, got %q", feed.Title)
	}
	if !strings.HasSuffix(feed.Link, "?q=rsshub") {
		t.Fatalf("search query should be appended to feed link, got %q", feed.Link)
	}
}

func TestTelegramRouteParamsParsing(t *testing.T) {
	opts := parseTelegramRouteParams("")
	if !opts.showLinkPreview || !opts.showMessageMedia || opts.searchQuery != "" {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
	opts = parseTelegramRouteParams("/showLinkPreview=0&showMessageMedia=false&searchQuery=rss")
	if opts.showLinkPreview || opts.showMessageMedia || opts.searchQuery != "rss" {
		t.Fatalf("unexpected parsed values: %+v", opts)
	}
	opts = parseTelegramRouteParams("/random tail")
	if opts.searchQuery != "random tail" || !opts.showLinkPreview || !opts.showMessageMedia {
		t.Fatalf("unknown tail should become searchQuery keeping defaults, got %+v", opts)
	}
}
