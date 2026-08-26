package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/parser"
	ctxpkg "github.com/xihale/syndi/pkg/context"
)

// withV2EXAPIBase points the V2EX JSON API base at baseURL for the duration of fn.
func withV2EXAPIBase(t *testing.T, baseURL string, fn func()) {
	t.Helper()
	orig := v2exAPIBase
	v2exAPIBase = baseURL
	defer func() { v2exAPIBase = orig }()
	fn()
}

func newV2EXFixtureServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		data, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Errorf("read fixture %s: %v", file, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func v2exContext(params map[string]string) *ctxpkg.Context {
	req := httptest.NewRequest(http.MethodGet, "/rss/test", nil)
	rec := httptest.NewRecorder()
	c := ctxpkg.NewContext(rec, req)
	c.SetParams(params)
	c.SetClient(client.New(client.WithTimeout(5 * time.Second)))
	return c
}

func TestV2EXTopicsLatest(t *testing.T) {
	srv := newV2EXFixtureServer(t, map[string]string{
		"/api/topics/latest.json": "topics_latest.json",
	})
	withV2EXAPIBase(t, srv.URL, func() {
		feed, err := V2EXTopicsHandler(v2exContext(map[string]string{"type": "latest"}))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(feed.Items); n != 2 {
			t.Fatalf("expected 2 topics, got %d", n)
		}
		first := feed.Items[0]
		if first.GUID != "v2ex-topic-1110001" || !strings.Contains(first.Link, "/t/1110001") {
			t.Fatalf("unexpected first topic item: %+v", first)
		}
		if first.Author == nil || first.Author.Name != "someone" {
			t.Fatalf("topic author missing: %+v", first)
		}
		if len(first.Categories) == 0 || first.Categories[0] != "python" {
			t.Fatalf("expected node name category, got %v", first.Categories)
		}
	})
}

func TestV2EXTopicsRejectsUnknownType(t *testing.T) {
	srv := newV2EXFixtureServer(t, nil)
	withV2EXAPIBase(t, srv.URL, func() {
		if _, err := V2EXTopicsHandler(v2exContext(map[string]string{"type": "everything"})); err == nil {
			t.Fatal("expected error for unknown topics type")
		}
	})
}

func TestV2EXPostAliasesTopicReplies(t *testing.T) {
	srv := newV2EXFixtureServer(t, map[string]string{
		"/api/topics/show.json":  "topic_detail.json",
		"/api/replies/show.json": "replies_list.json",
	})
	withV2EXAPIBase(t, srv.URL, func() {
		topicFeed, err := V2EXTopicHandler(v2exContext(map[string]string{"id": "584403"}))
		if err != nil {
			t.Fatal(err)
		}
		postFeed, err := V2EXPostHandler(v2exContext(map[string]string{"postid": "584403"}))
		if err != nil {
			t.Fatal(err)
		}

		if len(topicFeed.Items) == 0 || len(postFeed.Items) != len(topicFeed.Items) {
			t.Fatalf("post feed should mirror topic feed (%d vs %d)", len(postFeed.Items), len(topicFeed.Items))
		}
		for i := range postFeed.Items {
			got := postFeed.Items[i].GUID
			if !strings.HasPrefix(got, "v2ex-reply-") {
				t.Fatalf("reply GUID prefix mismatch: %q", got)
			}
			if got != topicFeed.Items[i].GUID {
				t.Fatalf("post and topic feeds diverge at item %d", i)
			}
			if postFeed.Items[i].Link == "" {
				t.Fatal("reply link missing")
			}
		}
	})
}

func TestParseV2EXTabItemFromFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "tab_items.html"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := parser.LoadString(string(data))
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	doc.Each("div.cell.item", func(_ int, sel *parser.Selection) {
		item := parseV2EXTabItem(sel)
		switch count {
		case 0:
			if item == nil {
				t.Fatal("expected first tab row to parse")
			}
			if item.Title != "mac m5 ultra 上线中国官网" {
				t.Fatalf("unexpected title: %q", item.Title)
			}
			if item.GUID != "v2ex-topic-1237246" {
				t.Fatalf("expected v2ex-topic GUID, got %q", item.GUID)
			}
			if item.Link != v2exBaseURL+"/t/1237246" {
				t.Fatalf("fragment should be stripped from link: %q", item.Link)
			}
			if len(item.Categories) != 1 || item.Categories[0] != "Local LLM" {
				t.Fatalf("expected node category, got %v", item.Categories)
			}
			if item.Author == nil || item.Author.Name != "someone" {
				t.Fatalf("expected author from member link, got %+v", item.Author)
			}
			if item.PubDate.IsZero() || item.PubDate.Year() != 2026 {
				t.Fatalf("absolute timestamp in title attribute should become pub date, got %v", item.PubDate)
			}
		case 1:
			if item == nil {
				t.Fatal("expected second tab row to parse")
			}
			if item.PubDate.IsZero() {
				t.Fatal("second row also carries an absolute timestamp")
			}
			if item.Author != nil {
				t.Fatalf("row without author must not invent one: %+v", item.Author)
			}
		default:
			if item != nil {
				t.Fatalf("malformed cell should not produce items: %+v", item)
			}
		}
		count++
	})

	if count != 3 { // plain separator div.cell (without .item) is not matched
		t.Fatalf("expected 3 div.cell.item rows in fixture, saw %d", count)
	}
}
