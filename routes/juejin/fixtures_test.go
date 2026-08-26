package routes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/routeutils"
)

func loadJuejinFixture(t *testing.T, name string) juejinResp {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var resp juejinResp
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	if err := resp.ok(); err != nil {
		t.Fatalf("fixture %s reports api error: %v", name, err)
	}
	return resp
}

func TestJuejinColumnDetailFixture(t *testing.T) {
	resp := loadJuejinFixture(t, "column_detail.json")
	detail, err := parseJuejinColumnDetail(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ColumnVersion.Title != "硬核JS" {
		t.Fatalf("title = %q", detail.ColumnVersion.Title)
	}
	if got := detail.intro(); got != "这里有那些你知道又不知道的硬核JS，由点及面，搞定JS" {
		t.Fatalf("intro = %q", got)
	}
	if detail.Author.UserName != "isboyjc" || detail.ColumnVersion.Cover == "" {
		t.Fatalf("author/cover not mapped: %+v %+v", detail.Author, detail.ColumnVersion)
	}
}

func TestJuejinColumnArticlesFixture(t *testing.T) {
	resp := loadJuejinFixture(t, "column_articles.json")
	var entries []juejinArticleEntry
	if err := json.Unmarshal(resp.Data, &entries); err != nil {
		t.Fatal(err)
	}

	feed := routeutils.NewFeed("test", "", "")
	mapJuejinEntries(feed, entries, "juejin-column-")
	if len(feed.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(feed.Items))
	}
	item := feed.Items[0]
	if item.GUID != "juejin-column-6997968693414084644" {
		t.Fatalf("guid = %q", item.GUID)
	}
	if item.Link != "https://juejin.cn/post/6997968693414084644" {
		t.Fatalf("link = %q", item.Link)
	}
	if item.Author == nil || item.Author.Name != "isboyjc" {
		t.Fatalf("author not mapped: %+v", item.Author)
	}
	cats := strings.Join(item.Categories, ",")
	for _, want := range []string{"前端", "JavaScript", "Promise"} {
		if !strings.Contains(cats, want) {
			t.Fatalf("categories missing %s: %v", want, item.Categories)
		}
	}
}

func TestJuejinTagDetailFixture(t *testing.T) {
	resp := loadJuejinFixture(t, "tag_detail.json")
	detail, err := parseJuejinTagDetail(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	if detail.TagID != "6809640398105870343" || detail.tagIcon() == "" {
		t.Fatalf("tag detail mismatch: id=%q icon=%q", detail.TagID, detail.tagIcon())
	}

	// top-level fallback when the nested tag object is absent
	fallback, err := parseJuejinTagDetail(json.RawMessage(`{"tag_id":"123","icon":"https://x/icon.png"}`))
	if err != nil {
		t.Fatal(err)
	}
	if fallback.TagID != "123" || fallback.tagIcon() != "https://x/icon.png" {
		t.Fatalf("fallback mismatch: %+v", fallback)
	}
}

func TestJuejinBooksFixture(t *testing.T) {
	resp := loadJuejinFixture(t, "books.json")
	books, err := parseJuejinBooks(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 booklet, got %d", len(books))
	}

	feed := routeutils.NewFeed("test", "", "")
	mapJuejinBooklets(feed, books)
	item := feed.Items[0]
	if item.GUID != "juejin-booklet-6844733720226856967" {
		t.Fatalf("guid = %q", item.GUID)
	}
	desc := item.Description
	if !strings.Contains(desc, `<img src="https://p1-juejin.byteimg.com/tos-cn-i-k3u1fbpfcp/book-cover.png"/>`) {
		t.Fatalf("cover missing: %q", desc)
	}
	if !strings.Contains(desc, "69.00 元") {
		t.Fatalf("price missing: %q", desc)
	}
	if strings.Contains(desc, "<script") {
		t.Fatalf("unescaped title leaked: %q", desc)
	}
}

func TestJuejinCollectionFixture(t *testing.T) {
	resp := loadJuejinFixture(t, "collection.json")
	col, err := parseJuejinCollection(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	if col.Detail.TagName != "前端收藏" || col.CreateUser.UserName != "mapoio" || len(col.ArticleList) != 1 {
		t.Fatalf("collection mismatch: %+v", col)
	}

	feed := routeutils.NewFeed(col.Detail.TagName+" - "+col.CreateUser.UserName+"的收藏集 - 掘金", "", "")
	mapJuejinEntries(feed, col.ArticleList, "juejin-collection-")
	if len(feed.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(feed.Items))
	}
	if feed.Items[0].GUID != "juejin-collection-6813709599467763726" {
		t.Fatalf("guid = %q", feed.Items[0].GUID)
	}
}

func TestJuejinDynamicFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "dynamic.json"))
	if err != nil {
		t.Fatal(err)
	}
	var resp juejinDynamicResp
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}
	list := resp.Data.List
	if len(list) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(list))
	}
	if list[0].User.UserName != "动态用户" {
		t.Fatalf("owner user not mapped: %+v", list[0].User)
	}

	feed := routeutils.NewFeedWithOptions(routeutils.FeedOptions{Title: "掘金用户动态-" + list[0].User.UserName})
	mapJuejinDynamicEntries(feed, list[0].User.UserName, list)

	// future_unknown_type must be skipped, other three mapped
	if len(feed.Items) != 3 {
		t.Fatalf("expected 3 items (unknown type skipped), got %d", len(feed.Items))
	}

	article := feed.Items[0]
	if article.Link != "https://juejin.cn/post/7678200303993405491" {
		t.Fatalf("article link = %q", article.Link)
	}
	if strings.Contains(article.Description, "<script") {
		t.Fatalf("unescaped brief leaked: %q", article.Description)
	}
	pin := feed.Items[1]
	if pin.Link != "https://juejin.cn/pin/7548882523352186915" {
		t.Fatalf("pin link = %q", pin.Link)
	}
	if !strings.Contains(pin.Description, `src="https://p1-juejin.byteimg.com/pic.png"`) {
		t.Fatalf("pin image missing: %q", pin.Description)
	}
	if len(pin.Categories) != 1 || pin.Categories[0] != "上班摸鱼" {
		t.Fatalf("pin topic category missing: %v", pin.Categories)
	}
	follow := feed.Items[2]
	if follow.Title != "动态用户 关注了 mapoio" {
		t.Fatalf("follow title = %q", follow.Title)
	}
}
