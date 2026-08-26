package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/client"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// youTubeFixtureServer serves recorded official-endpoint payloads locally:
//   - /feeds/videos.xml?...  -> testdata/channel_feed.atom
//   - /@<handle> and /c/<name> -> testdata/handle_page.html
//
// The last requested feed query string is stored in lastFeedQuery so tests can
// assert which endpoint variant the handler chose.
func youTubeFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/feeds/videos.xml"):
			lastFeedQueryValue = r.URL.RawQuery
			http.ServeFile(w, r, "testdata/channel_feed.atom")
		case strings.HasPrefix(r.URL.Path, "/@"), strings.HasPrefix(r.URL.Path, "/c/"):
			http.ServeFile(w, r, "testdata/handle_page.html")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func withYouTubeFixtures(t *testing.T) *httptest.Server {
	t.Helper()
	server := youTubeFixtureServer(t)
	prevFeed, prevPage := youTubeFeedDataURL, youTubePageDataURL
	youTubeFeedDataURL = server.URL
	youTubePageDataURL = server.URL
	t.Cleanup(func() {
		youTubeFeedDataURL = prevFeed
		youTubePageDataURL = prevPage
	})
	return server
}

func runYouTubeHandler(t *testing.T, handler func(*ctxpkg.Context) (*models.Feed, error), params map[string]string) (*models.Feed, error) {
	t.Helper()
	withYouTubeFixtures(t)
	req := httptest.NewRequest(http.MethodGet, "/rss-test", nil)
	c := ctxpkg.NewContext(httptest.NewRecorder(), req)
	c.SetParams(params)
	c.SetClient(client.New(client.WithTimeout(10 * time.Second)))
	return handler(c)
}

// lastFeedQueryValue records the most recent /feeds/videos.xml query so tests
// can assert which endpoint variant the handler picked.
var lastFeedQueryValue string

func readFixtureCount(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile("testdata/channel_feed.atom")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(data), "<entry>")
}

func TestYouTubeChannelFixture(t *testing.T) {
	feed, err := runYouTubeHandler(t, YouTubeChannelHandler, map[string]string{"id": "UCX6OQ3DkcsbYNE6H8uQQuVA"})
	if err != nil {
		t.Fatal(err)
	}
	if want := readFixtureCount(t); len(feed.Items) != want {
		t.Fatalf("expected %d items, got %d", want, len(feed.Items))
	}
	if feed.Title != "MrBeast" {
		t.Fatalf("unexpected feed title %q", feed.Title)
	}
	first := feed.Items[0]
	if !strings.HasPrefix(first.GUID, "yt:video:") {
		t.Fatalf("expected native yt:video GUID, got %q", first.GUID)
	}
	if !strings.Contains(first.Description, "<iframe") || !strings.Contains(first.Description, "/embed/") {
		t.Fatalf("expected embedded player by default, got %.200q", first.Description)
	}
}

func TestYouTubeChannelFixtureSwitches(t *testing.T) {
	feed, err := runYouTubeHandler(t, YouTubeChannelHandler, map[string]string{
		"id":          "UCX6OQ3DkcsbYNE6H8uQQuVA",
		"routeParams": "/filterShorts=1&embed=0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if q := lastFeedQueryValue; !strings.Contains(q, "playlist_id=UULFX6OQ3DkcsbYNE6H8uQQuVA") {
		t.Fatalf("expected shorts-free UULF playlist query, got %q", q)
	}
	for _, item := range feed.Items {
		if strings.Contains(item.Description, "<iframe") {
			t.Fatalf("unexpected iframe with embed=0 in %q", item.GUID)
		}
		if !strings.Contains(item.Description, "<img src=") {
			t.Fatalf("expected thumbnail image with embed=0 in %q", item.GUID)
		}
	}
}

func TestYouTubeUserLegacyUsernameFixture(t *testing.T) {
	feed, err := runYouTubeHandler(t, YouTubeUserHandler, map[string]string{"username": "Google"})
	if err != nil {
		t.Fatal(err)
	}
	if q := lastFeedQueryValue; !strings.HasPrefix(q, "user=Google") {
		t.Fatalf("legacy usernames should use the official ?user= feed, got %q", q)
	}
	if len(feed.Items) != readFixtureCount(t) {
		t.Fatalf("unexpected item count %d", len(feed.Items))
	}
}

func TestYouTubeUserHandleFixture(t *testing.T) {
	feed, err := runYouTubeHandler(t, YouTubeUserHandler, map[string]string{
		"username":    "@JFlaMusic",
		"routeParams": "/filterShorts=1",
	})
	if err != nil {
		t.Fatal(err)
	}
	// filterShorts forces handle resolution then the UULF uploads playlist.
	if q := lastFeedQueryValue; !strings.Contains(q, "playlist_id=UULFSJ4gkVC6NrvII8umztf0Ow") {
		t.Fatalf("expected UULF playlist from resolved handle, got %q", q)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
}

func TestYouTubeCustomURLFixture(t *testing.T) {
	feed, err := runYouTubeHandler(t, YouTubeCustomHandler, map[string]string{"username": "TED"})
	if err != nil {
		t.Fatal(err)
	}
	// The /c/ probe reuses the recorded channel-page HTML whose externalId is
	// the handle fixture's channel.
	if q := lastFeedQueryValue; !strings.Contains(q, "channel_id=UCSJ4gkVC6NrvII8umztf0Ow") {
		t.Fatalf("expected resolved /c/ channel id in query, got %q", q)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
}

func TestYouTubePlaylistEmbedOffFixture(t *testing.T) {
	feed, err := runYouTubeHandler(t, YouTubePlaylistHandler, map[string]string{
		"id":          "PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z",
		"routeParams": "/embed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if q := lastFeedQueryValue; !strings.HasPrefix(q, "playlist_id=PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z") {
		t.Fatalf("unexpected playlist query %q", q)
	}
	for _, item := range feed.Items {
		if strings.Contains(item.Description, "<iframe") {
			t.Fatalf("bare /embed should disable iframes in %q", item.GUID)
		}
	}
}

func TestYouTubePlaylistInvalidID(t *testing.T) {
	_, err := runYouTubeHandler(t, YouTubePlaylistHandler, map[string]string{"id": "watch?v=x"})
	if err == nil || !strings.Contains(err.Error(), "invalid YouTube playlist id") {
		t.Fatalf("expected invalid playlist id error, got %v", err)
	}
}

func TestYouTubeRouteParamsParsing(t *testing.T) {
	p := parseYouTubeRouteParams("")
	if !p.embed || p.filterShorts {
		t.Fatalf("expected embedding on and shorts unfiltered by default: %+v", p)
	}
	p = parseYouTubeRouteParams("/embed")
	if p.embed {
		t.Fatalf("bare embed should disable embedding: %+v", p)
	}
	p = parseYouTubeRouteParams("/embed=0&filterShorts=true")
	if p.embed != false || p.filterShorts != true {
		t.Fatalf("unexpected parsed values: %+v", p)
	}
	p = parseYouTubeRouteParams("/embed=1&filterShorts=false")
	if p.embed != true || p.filterShorts != false {
		t.Fatalf("unexpected explicit values: %+v", p)
	}
}
