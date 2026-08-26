package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
)

// withWeiboAPIBase points the m.weibo.cn origin at baseURL for fn's duration.
func withWeiboAPIBase(t *testing.T, baseURL string, fn func()) {
	t.Helper()
	orig := weiboMobileAPIBase
	weiboMobileAPIBase = baseURL
	t.Cleanup(func() { weiboMobileAPIBase = orig })
	fn()
}

// newWeiboJSONFixtureServer serves one JSON fixture on path.
func newWeiboJSONFixtureServer(t *testing.T, file string) *httptest.Server {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		t.Fatalf("read fixture %s: %v", file, err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newOfflineContext builds a handler context that never needs real network.
func newOfflineContext(params map[string]string) *ctxpkg.Context {
	req := httptest.NewRequest(http.MethodGet, "/rss/test", nil)
	rec := httptest.NewRecorder()
	c := ctxpkg.NewContext(rec, req)
	if len(params) > 0 {
		c.SetParams(params)
	}
	c.SetClient(client.New(client.WithTimeout(5 * time.Second)))
	return c
}

func TestWeiboKeywordFromFixture(t *testing.T) {
	srv := newWeiboJSONFixtureServer(t, "keyword_cards.json")
	withWeiboAPIBase(t, srv.URL, func() {
		feed, err := WeiboKeywordHandler(newOfflineContext(map[string]string{"keyword": "RSSHub"}))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(feed.Items); n != 2 {
			t.Fatalf("expected 2 items (mblog-less card skipped), got %d", n)
		}
		first := feed.Items[0]
		if !strings.HasPrefix(first.GUID, "weibo-keyword-") {
			t.Fatalf("unexpected guid %q", first.GUID)
		}
		if first.Link != "https://weibo.com/1195230310/ABCdef12" {
			t.Fatalf("unexpected link %q", first.Link)
		}
		if first.Author == nil || first.Author.Name != "开源观察" {
			t.Fatalf("expected author 开源观察, got %+v", first.Author)
		}
		if first.PubDate.IsZero() {
			t.Fatal("absolute created_at should parse")
		}
		second := feed.Items[1]
		if got := second.PubDate; got.IsZero() || time.Since(got) > 3*time.Hour {
			t.Fatalf("relative created_at should resolve near now, got %v", second.PubDate)
		}
	})
}

func TestWeiboSuperIndexFromFixture(t *testing.T) {
	srv := newWeiboJSONFixtureServer(t, "super_index_cards.json")
	withWeiboAPIBase(t, srv.URL, func() {
		feed, err := WeiboSuperIndexHandler(newOfflineContext(map[string]string{"id": "1008084989d223732bf6f02f75ea30efad58a9"}))
		if err != nil {
			t.Fatal(err)
		}
		if feed.Title != "微博超话 - Rsshub超话" {
			t.Fatalf("unexpected feed title %q", feed.Title)
		}
		if len(feed.Items) != 2 {
			t.Fatalf("expected nested card_group mblogs collected, got %d items", len(feed.Items))
		}
		first := feed.Items[0]
		if !strings.HasPrefix(first.GUID, "weibo-super-index-") {
			t.Fatalf("unexpected guid %q", first.GUID)
		}
		if !strings.HasPrefix(first.Link, "https://weibo.com/3333333333/") {
			t.Fatalf("author-id permalink fallback expected, got %q", first.Link)
		}
	})
}

// The search/hot paths must reuse WeiboHotSearchHandler (identical upstream
// container API) with the hot-board TTL kept within 10 minutes.
func TestWeiboSearchHotRoutesPointAtHotSearchHandler(t *testing.T) {
	want := reflect.ValueOf(WeiboHotSearchHandler).Pointer()
	for _, tc := range []struct {
		name string
		spec routeutils.RouteSpec
	}{
		{"search/hot", weiboSearchHotRoute},
		{"search/hot/:fulltext", weiboSearchHotFulltextRoute},
	} {
		if tc.spec.Handler == nil || reflect.ValueOf(tc.spec.Handler).Pointer() != want {
			t.Fatalf("%s should share WeiboHotSearchHandler", tc.name)
		}
		if tc.spec.CacheTTL > 10*time.Minute {
			t.Fatalf("%s cache TTL should stay within 10 minutes, got %v", tc.name, tc.spec.CacheTTL)
		}
	}
	if weiboHotSearchRoute.CacheTTL != 10*time.Minute {
		t.Fatalf("hotsearch board TTL should be 10 minutes, got %v", weiboHotSearchRoute.CacheTTL)
	}
}
