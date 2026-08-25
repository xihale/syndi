package routeutils

import (
	"testing"
	"time"

	"github.com/xihale/syndi/pkg/cache"
	"github.com/xihale/syndi/pkg/models"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		rel     string
		want    string
		wantErr bool
	}{
		{
			name: "absolute URL",
			base: "https://example.com/",
			rel:  "https://other.com/path",
			want: "https://other.com/path",
		},
		{
			name: "relative URL",
			base: "https://example.com/base/",
			rel:  "path/to/resource",
			want: "https://example.com/base/path/to/resource",
		},
		{
			name: "protocol-relative URL",
			base: "https://example.com/",
			rel:  "//cdn.example.com/script.js",
			want: "https://cdn.example.com/script.js",
		},
		{
			name: "root-relative URL",
			base: "https://example.com/base/",
			rel:  "/root/path",
			want: "https://example.com/root/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveURL(tt.base, tt.rel)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFeed(t *testing.T) {
	feed := NewFeed("Test Feed", "https://example.com", "Test Description")

	if feed.Title != "Test Feed" {
		t.Errorf("NewFeed() Title = %v, want %v", feed.Title, "Test Feed")
	}
	if feed.Link != "https://example.com" {
		t.Errorf("NewFeed() Link = %v, want %v", feed.Link, "https://example.com")
	}
	if feed.Description != "Test Description" {
		t.Errorf("NewFeed() Description = %v, want %v", feed.Description, "Test Description")
	}
	if feed.Items == nil {
		t.Errorf("NewFeed() Items should be initialized")
	}
}

func TestNewItem(t *testing.T) {
	pubDate := time.Now()
	item := NewItem("Test Item", "https://example.com/item", "Description", pubDate)

	if item.Title != "Test Item" {
		t.Errorf("NewItem() Title = %v, want %v", item.Title, "Test Item")
	}
	if item.Link != "https://example.com/item" {
		t.Errorf("NewItem() Link = %v, want %v", item.Link, "https://example.com/item")
	}
	if item.GUID != "https://example.com/item" {
		t.Errorf("NewItem() GUID = %v, want %v", item.GUID, "https://example.com/item")
	}
	if item.PubDate != pubDate {
		t.Errorf("NewItem() PubDate = %v, want %v", item.PubDate, pubDate)
	}
}

func TestAddItem(t *testing.T) {
	feed := NewFeed("Test", "https://example.com", "Desc")
	item := NewItem("Item", "https://example.com/item", "Desc", time.Now())

	AddItem(feed, item)

	if len(feed.Items) != 1 {
		t.Errorf("AddItem() feed.Items length = %v, want %v", len(feed.Items), 1)
	}
}

func TestSetCategories(t *testing.T) {
	item := &models.Item{}

	SetCategories(item, "cat1", "cat2", "cat3")

	if len(item.Categories) != 3 {
		t.Errorf("SetCategories() categories length = %v, want %v", len(item.Categories), 3)
	}
}

func TestGetAuthorString(t *testing.T) {
	tests := []struct {
		name string
		item *models.Item
		want string
	}{
		{
			name: "with author",
			item: &models.Item{
				Author: &models.Author{Name: "John Doe"},
			},
			want: "John Doe",
		},
		{
			name: "nil author",
			item: &models.Item{Author: nil},
			want: "",
		},
		{
			name: "nil item",
			item: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAuthorString(tt.item); got != tt.want {
				t.Errorf("GetAuthorString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetAuthor(t *testing.T) {
	item := &models.Item{}

	SetAuthor(item, "Jane Doe", WithAuthorEmail("jane@example.com"))

	if item.Author.Name != "Jane Doe" {
		t.Errorf("SetAuthor() Name = %v, want %v", item.Author.Name, "Jane Doe")
	}
	if item.Author.Email != "jane@example.com" {
		t.Errorf("SetAuthor() Email = %v, want %v", item.Author.Email, "jane@example.com")
	}
}

func TestFilterByTime(t *testing.T) {
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	recent := now.Add(-30 * time.Minute)

	items := []models.Item{
		{Title: "Old", PubDate: old},
		{Title: "Recent", PubDate: recent},
	}

	// Filter to last hour (3600 seconds)
	filtered := FilterByTime(items, 3600)

	if len(filtered) != 1 {
		t.Errorf("FilterByTime() filtered length = %v, want %v", len(filtered), 1)
	}
	if filtered[0].Title != "Recent" {
		t.Errorf("FilterByTime() first item = %v, want %v", filtered[0].Title, "Recent")
	}
}

func TestSortByPubDate(t *testing.T) {
	now := time.Now()
	items := []models.Item{
		{Title: "First", PubDate: now.Add(-2 * time.Hour)},
		{Title: "Third", PubDate: now.Add(-3 * time.Hour)},
		{Title: "Second", PubDate: now.Add(-1 * time.Hour)},
	}

	SortByPubDate(items, true) // descending (newest first)

	// Newest is -1 hour (Second), oldest is -3 hours (Third)
	if items[0].Title != "Second" {
		t.Errorf("SortByPubDate() first item = %v, want %v", items[0].Title, "Second")
	}
	if items[2].Title != "Third" {
		t.Errorf("SortByPubDate() last item = %v, want %v", items[2].Title, "Third")
	}
}

func TestApplyLimit(t *testing.T) {
	items := make([]models.Item, 10)
	for i := range items {
		items[i] = models.Item{Title: string(rune('A' + i))}
	}

	limited := ApplyLimit(items, 5)

	if len(limited) != 5 {
		t.Errorf("ApplyLimit() length = %v, want %v", len(limited), 5)
	}
}

func TestCacheFeed(t *testing.T) {
	cache := cache.NewMemoryCache(100)
	key := "test-feed"
	ttl := 5 * time.Minute

	callCount := 0
	fetchFn := func() (*models.Feed, error) {
		callCount++
		return NewFeed("Test", "https://example.com", "Desc"), nil
	}

	// First call - cache miss
	feed1, err := CacheFeed(cache, key, ttl, fetchFn)
	if err != nil {
		t.Fatalf("CacheFeed() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("CacheFeed() callCount = %v, want %v", callCount, 1)
	}

	// Second call - cache hit: the fetch function must NOT run again.
	feed2, err := CacheFeed(cache, key, ttl, fetchFn)
	if err != nil {
		t.Fatalf("CacheFeed() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("CacheFeed() callCount after cache hit = %v, want %v", callCount, 1)
	}

	if feed1.Title != feed2.Title {
		t.Errorf("CacheFeed() feed titles don't match: %v vs %v", feed1.Title, feed2.Title)
	}
}

func TestAddQueryParam(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		key     string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:  "add to URL without params",
			url:   "https://example.com",
			key:   "param",
			value: "value",
			want:  "https://example.com?param=value",
		},
		{
			name:  "add to URL with existing params",
			url:   "https://example.com?existing=1",
			key:   "param",
			value: "value",
			want:  "https://example.com?existing=1&param=value",
		},
		{
			name:  "replace existing param",
			url:   "https://example.com?param=old",
			key:   "param",
			value: "new",
			want:  "https://example.com?param=new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AddQueryParam(tt.url, tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddQueryParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AddQueryParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStripQuery(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "URL with query",
			url:  "https://example.com?param=value",
			want: "https://example.com",
		},
		{
			name: "URL with fragment",
			url:  "https://example.com#section",
			want: "https://example.com",
		},
		{
			name: "URL with both",
			url:  "https://example.com?param=value#section",
			want: "https://example.com",
		},
		{
			name: "plain URL",
			url:  "https://example.com/path",
			want: "https://example.com/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripQuery(tt.url); got != tt.want {
				t.Errorf("StripQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "remove fragment",
			url:  "https://example.com#section",
			want: "https://example.com",
		},
		{
			name: "sort query params",
			url:  "https://example.com?z=1&a=2&b=3",
			want: "https://example.com?a=2&b=3&z=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCleanDescription(t *testing.T) {
	opts := DefaultCleanOptions()

	t.Run("keeps entity-escaped text escaped", func(t *testing.T) {
		html := `<p>Hello &amp; world</p>`
		cleaned, err := CleanDescription(html, "https://example.com", opts)
		if err != nil {
			t.Fatalf("CleanDescription() error = %v", err)
		}
		if !contains(cleaned, "Hello &amp; world") {
			t.Errorf("CleanDescription() = %q, want entity-escaped text preserved", cleaned)
		}
		if contains(cleaned, "<html") || contains(cleaned, "<body") {
			t.Errorf("CleanDescription() = %q, must not wrap fragments in html/body", cleaned)
		}
	})

	t.Run("does not activate escaped markup", func(t *testing.T) {
		// Escaped markup in source must stay escaped: decoding entities
		// before the HTML passes below would turn this into a live tag.
		html := `&lt;img src=x onerror=alert(1)&gt;`
		cleaned, err := CleanDescription(html, "https://example.com", opts)
		if err != nil {
			t.Fatalf("CleanDescription() error = %v", err)
		}
		if contains(cleaned, "<img") {
			t.Errorf("CleanDescription() = %q, escaped markup was activated", cleaned)
		}
	})
}

func TestSanitizeHTMLAttributes(t *testing.T) {
	// Event handlers and script-capable URL schemes must be filtered even
	// when the tag itself is allowed.
	in := `<a href="javascript:alert(1)" onclick="steal()" title="ok">c</a>`
	got, err := SanitizeHTML(in, []string{"a"}, nil)
	if err != nil {
		t.Fatalf("SanitizeHTML() error = %v", err)
	}
	if contains(got, "onclick") || contains(got, "javascript:") {
		t.Errorf("SanitizeHTML() = %q, dangerous attributes survived", got)
	}
	if !contains(got, `title="ok"`) || !contains(got, ">c</a>") {
		t.Errorf("SanitizeHTML() = %q, safe attributes/content must survive", got)
	}
}

func TestStripHTMLRemovesScriptContent(t *testing.T) {
	if got := ExtractText("<script>var a=1;</script>ok"); got != "ok" {
		t.Errorf("ExtractText() = %q, want %q (script body must not leak)", got, "ok")
	}
}

func TestDecodeHTMLEntities(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "ampersand",
			s:    "Hello &amp; world",
			want: "Hello & world",
		},
		{
			name: "less than",
			s:    "&lt;tag&gt;",
			want: "<tag>",
		},
		{
			name: "quote",
			s:    "&quot;quoted&quot;",
			want: "\"quoted\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DecodeHTMLEntities(tt.s); got != tt.want {
				t.Errorf("DecodeHTMLEntities() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		suffix    string
		want      string
	}{
		{
			name:      "no truncation needed",
			text:      "Short",
			maxLength: 20,
			suffix:    "...",
			want:      "Short",
		},
		{
			name:      "truncate needed",
			text:      "This is a very long text that needs truncation",
			maxLength: 20,
			suffix:    "...",
			want:      "This is a very long...", // Truncate finds the last space within the limit
		},
		{
			name:      "truncate at word boundary",
			text:      "This is a sample text that is too long",
			maxLength: 15,
			suffix:    "...",
			want:      "This is a...", // Last space before position 15 is after "a"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.text, tt.maxLength, tt.suffix); got != tt.want {
				t.Errorf("Truncate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUniqueByLink(t *testing.T) {
	items := []models.Item{
		{Link: "https://example.com/1", Title: "First"},
		{Link: "https://example.com/2", Title: "Second"},
		{Link: "https://example.com/1", Title: "Duplicate"},
	}

	unique := UniqueByLink(items)

	if len(unique) != 2 {
		t.Errorf("UniqueByLink() length = %v, want %v", len(unique), 2)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "single part",
			parts: []string{"feed"},
			want:  "feed",
		},
		{
			name:  "multiple parts",
			parts: []string{"feed", "github", "repos"},
			want:  "feed:github:repos",
		},
		{
			name:  "empty parts",
			parts: []string{"feed", "", "repos"},
			want:  "feed:repos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateCacheKey(tt.parts...); got != tt.want {
				t.Errorf("GenerateCacheKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
