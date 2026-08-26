package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/client"
	ctxpkg "github.com/xihale/syndi/pkg/context"
)

// withITHomeRoot points the site origin at the fixture server for fn's
// duration so handlers fetch fixture HTML instead of ithome.com.
func withITHomeRoot(t *testing.T, baseURL string, fn func()) {
	t.Helper()
	orig := ithomeRoot
	ithomeRoot = baseURL
	t.Cleanup(func() { ithomeRoot = orig })
	fn()
}

// newITHomeFixtureServer maps request paths to testdata files, substituting
// {ROOT} with the server origin. Missing paths return 404 to exercise the
// best-effort detail fetch fallback.
func newITHomeFixtureServer(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()
	type entry struct {
		data []byte
	}
	entries := make(map[string]*entry)
	for path, file := range files {
		data, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Fatalf("read fixture %s: %v", file, err)
		}
		entries[path] = &entry{data: data}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e, ok := entries[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(strings.ReplaceAll(string(e.data), "{ROOT}", ithomeRoot)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newITHomeContext builds a handler context against a (usually synthetic)
// request URL so query params like limit can be exercised.
func newITHomeContext(requestURL string, params map[string]string) *ctxpkg.Context {
	req := httptest.NewRequest(http.MethodGet, requestURL, nil)
	rec := httptest.NewRecorder()
	c := ctxpkg.NewContext(rec, req)
	if len(params) > 0 {
		c.SetParams(params)
	}
	c.SetClient(client.New(client.WithTimeout(5 * time.Second)))
	return c
}

func TestITHomeRankingFromFixture(t *testing.T) {
	srv := newITHomeFixtureServer(t, map[string]string{
		"/block/rank.html": "rank.html",
		"/0/994/490.htm":   "article.html",
		"/0/994/328.htm":   "article.html",
	})
	withITHomeRoot(t, srv.URL, func() {
		feed, err := ITHomeRankingHandler(newITHomeContext("/rss/test", map[string]string{"type": "24h"}))
		if err != nil {
			t.Fatal(err)
		}
		if feed.Title != "IT之家-24 小时最热" {
			t.Fatalf("unexpected feed title %q", feed.Title)
		}
		if len(feed.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(feed.Items))
		}
		first := feed.Items[0]
		if first.GUID != "ithome-ranking-0/994/490.htm" {
			t.Fatalf("unexpected guid %q", first.GUID)
		}
		want := time.Date(2026, 8, 26, 14, 23, 15, 0, ithomeCSTZone)
		if !first.PubDate.Equal(want) {
			t.Fatalf("expected pubtime %v, got %v", want, first.PubDate)
		}
		if first.Author == nil || first.Author.Name != "故渊" {
			t.Fatalf("expected author 故渊, got %+v", first.Author)
		}
		if !strings.Contains(first.Description, "正文首段") || strings.Contains(first.Description, "alert(") {
			t.Fatalf("description should keep body text but drop scripts")
		}
		if !strings.Contains(first.Description, `/img/upload/lazy.jpg`) {
			t.Fatalf("lazy image data-original should be promoted to src")
		}
	})
}

func TestITHomeRankingInvalidType(t *testing.T) {
	if _, err := ITHomeRankingHandler(newITHomeContext("/rss/test", map[string]string{"type": "week"})); err == nil ||
		!strings.Contains(err.Error(), "24h") {
		t.Fatalf("expected actionable error for unknown type, got %v", err)
	}
}

func TestITHomeTagFeed(t *testing.T) {
	srv := newITHomeFixtureServer(t, map[string]string{
		"/tag/win11":     "tag_win11.html",
		"/0/994/535.htm": "article.html",
	})
	withITHomeRoot(t, srv.URL, func() {
		feed, err := ITHomeTagHandler(newITHomeContext("/rss/test?limit=2", map[string]string{"name": "win11"}))
		if err != nil {
			t.Fatal(err)
		}
		if feed.Title != "IT之家 - win11标签" {
			t.Fatalf("unexpected feed title %q", feed.Title)
		}
		if len(feed.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(feed.Items))
		}
		first := feed.Items[0]
		if first.GUID != "ithome-tag-0/994/535.htm" {
			t.Fatalf("unexpected guid %q", first.GUID)
		}
		want := time.Date(2026, 8, 26, 14, 23, 15, 0, ithomeCSTZone)
		if !first.PubDate.Equal(want) {
			t.Fatalf("data-ot timestamp should parse, got %v", first.PubDate)
		}
		second := feed.Items[1]
		if second.Description != "<p>摘要缺失兜底。</p>" {
			t.Fatalf("failed detail fetch should fall back to summary, got %q", second.Description)
		}
		// Its invalid data-ot and missing article page leave the timestamp zero.
		if !second.PubDate.IsZero() {
			t.Fatalf("invalid data-ot with missing article should keep zero time, got %v", second.PubDate)
		}
	})
}

func TestITHomeZTFromFixtureDefaultID(t *testing.T) {
	srv := newITHomeFixtureServer(t, map[string]string{
		"/zt/xijiayi": "zt_xijiayi.html",
	})
	withITHomeRoot(t, srv.URL, func() {
		feed, err := ITHomeZTHandler(newITHomeContext("/rss/test", nil))
		if err != nil {
			t.Fatal(err)
		}
		if feed.Title != "IT之家 - 「喜加一」最新动态" {
			t.Fatalf("unexpected feed title %q", feed.Title)
		}
		if len(feed.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(feed.Items))
		}
		first := feed.Items[0]
		if first.GUID != "ithome-zt-0/994/308.htm" {
			t.Fatalf("unexpected guid %q", first.GUID)
		}
		if first.Author == nil || first.Author.Name != "漾仔" {
			t.Fatalf("expected list-level author 漾仔, got %+v", first.Author)
		}
		want := time.Date(2026, 8, 26, 7, 30, 22, 0, ithomeCSTZone)
		if !first.PubDate.Equal(want) {
			t.Fatalf("jsDateDiff literal should parse, got %v", first.PubDate)
		}
		if second := feed.Items[1]; second.Link != fmt.Sprintf("%s/0/994/309.htm", ithomeRoot) {
			t.Fatalf("unexpected second link %q", second.Link)
		}
	})
}

func TestITHomeGUIDPath(t *testing.T) {
	if got := ithomeGUIDPath("https://www.ithome.com/0/994/535.htm"); got != "0/994/535.htm" {
		t.Fatalf("unexpected path %q", got)
	}
}
