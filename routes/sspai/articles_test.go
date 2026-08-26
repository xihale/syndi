package routes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
)

func loadSspaiFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestSspaiArticlesListFixture(t *testing.T) {
	var list sspaiListResp
	if err := json.Unmarshal(loadSspaiFixture(t, "articles_list.json"), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.List) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(list.List))
	}

	feed := routeutils.NewFeed("test", "https://sspai.com/", "test")
	mapSspaiArticles(feed, list.List, "sspai-matrix-")

	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.GUID != "sspai-matrix-113626" {
		t.Fatalf("guid = %q, want prefixed id", first.GUID)
	}
	if first.Link != "https://sspai.com/post/113626" {
		t.Fatalf("link = %q", first.Link)
	}
	if first.Author == nil || first.Author.Name != "onevcat" {
		t.Fatalf("author not mapped: %+v", first.Author)
	}
	want := time.Unix(1787535788, 0)
	if !first.PubDate.Equal(want) {
		t.Fatalf("pubdate = %v, want %v", first.PubDate, want)
	}

	// XSS-safe descriptions: summaries must be HTML-escaped
	for _, item := range feed.Items {
		if strings.Contains(item.Description, "<script") || strings.Contains(item.Description, "<img") {
			t.Fatalf("unescaped content leaked: %q", item.Description)
		}
	}
	if got := feed.Items[1].Description; !strings.Contains(got, "&lt;img") || !strings.Contains(got, "&gt;") {
		t.Fatalf("escaped summary expected, got %q", got)
	}
}

func TestSspaiIndexPageFixture(t *testing.T) {
	var resp sspaiWrapped[[]sspaiArticle]
	if err := json.Unmarshal(loadSspaiFixture(t, "index_page.json"), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != 0 || len(resp.Data) != 1 {
		t.Fatalf("bad envelope: error=%d n=%d", resp.Error, len(resp.Data))
	}

	feed := routeutils.NewFeed("test", "", "")
	mapSspaiArticles(feed, resp.Data, "sspai-index-")
	item := feed.Items[0]
	if item.GUID != "sspai-index-68227" {
		t.Fatalf("guid = %q", item.GUID)
	}
	if item.PubDate.IsZero() {
		t.Fatal("released_time variant should still populate PubDate")
	}
	if item.Author == nil || item.Author.Name != "jijiali" {
		t.Fatalf("author not mapped: %+v", item.Author)
	}
}

func TestSspaiSpecialColumnFixture(t *testing.T) {
	var resp sspaiWrapped[sspaiSpecialColumn]
	if err := json.Unmarshal(loadSspaiFixture(t, "special_column.json"), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != 0 {
		t.Fatalf("unexpected error code %d", resp.Error)
	}
	if resp.Data.Title != "GoodNotes" || resp.Data.Intro != "书写一切" {
		t.Fatalf("column detail mismatch: %+v", resp.Data)
	}
}

func TestSspaiTopicsFixture(t *testing.T) {
	var list sspaiListTopics
	if err := json.Unmarshal(loadSspaiFixture(t, "topics.json"), &list); err != nil {
		t.Fatal(err)
	}

	feed := routeutils.NewFeed("test", "", "")
	mapSspaiTopics(feed, list.List)
	if len(feed.Items) != 1 {
		t.Fatalf("expected 1 topic item, got %d", len(feed.Items))
	}
	item := feed.Items[0]
	if item.GUID != "sspai-topic-527" {
		t.Fatalf("guid = %q", item.GUID)
	}
	desc := item.Description
	if !strings.Contains(desc, `<img src="`+sspaiCDNPrefix+"8/13/2026/article/") {
		t.Fatalf("banner missing CDN prefix: %q", desc)
	}
	if !strings.Contains(desc, "开始你的少数派创作之旅") {
		t.Fatalf("intro missing: %q", desc)
	}
}

func TestSspaiBookmarksFixture(t *testing.T) {
	var favs sspaiWrapped[[]sspaiArticle]
	if err := json.Unmarshal(loadSspaiFixture(t, "favorites_page.json"), &favs); err != nil {
		t.Fatal(err)
	}
	var user sspaiWrapped[sspaiSlugInfo]
	if err := json.Unmarshal(loadSspaiFixture(t, "slug_info.json"), &user); err != nil {
		t.Fatal(err)
	}
	if user.Data.Nickname != "文刀行人" || user.Data.ID != 997539 {
		t.Fatalf("slug info mismatch: %+v", user.Data)
	}

	feed := routeutils.NewFeed(user.Data.Nickname+" 的全部收藏 - 少数派", "", "")
	mapSspaiArticles(feed, favs.Data, "sspai-bookmark-")
	if len(feed.Items) != 1 {
		t.Fatalf("expected 1 bookmark item, got %d", len(feed.Items))
	}
	if feed.Items[0].GUID != "sspai-bookmark-78322" {
		t.Fatalf("guid = %q", feed.Items[0].GUID)
	}
}

func TestIsSspaiNumeric(t *testing.T) {
	for raw, want := range map[string]bool{"796518": true, "": false, "urfp0d9i": false, "-5": false} {
		if got := isSSPAINumeric(raw); got != want {
			t.Fatalf("isSSPAINumeric(%q) = %v, want %v", raw, got, want)
		}
	}
}
