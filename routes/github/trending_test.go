package routes

import (
	"testing"
	"time"

	"github.com/xihale/syndi/internal/parser"
)

func TestParseGitHubTrendingItem(t *testing.T) {
	html := `
<article class="Box-row">
  <h2><a href="/owner/repo"> owner / repo </a></h2>
  <p> Sample description </p>
  <a href="/owner/repo/stargazers">1,234</a>
  <a href="/owner/repo/forks">88</a>
  <span itemprop="programmingLanguage"> Go </span>
  <span>123 stars today</span>
</article>`

	doc, err := parser.LoadString(html)
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	now := time.Date(2026, time.February, 22, 10, 0, 0, 0, time.UTC)
	item := parseGitHubTrendingItem(doc.First("article.Box-row"), now)
	if item == nil {
		t.Fatal("expected non-nil item")
	}

	if item.Title != "owner / repo" {
		t.Fatalf("unexpected title: %q", item.Title)
	}
	if item.Link != "https://github.com/owner/repo" {
		t.Fatalf("unexpected link: %q", item.Link)
	}
	if item.GUID != item.Link {
		t.Fatalf("expected GUID to equal link, got %q", item.GUID)
	}
	if item.PubDate != now {
		t.Fatalf("unexpected pub date: %v", item.PubDate)
	}

	wantDescription := "Sample description<br/>Stats: 1,234 | 88"
	if item.Description != wantDescription {
		t.Fatalf("unexpected description: %q", item.Description)
	}

	if len(item.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d (%v)", len(item.Categories), item.Categories)
	}
	if item.Categories[0] != "Go" || item.Categories[1] != "123 stars today" {
		t.Fatalf("unexpected categories: %v", item.Categories)
	}
}

func TestParseGitHubTrendingItemRequiresTitleAndLink(t *testing.T) {
	html := `<article class="Box-row"><h2><a>no link</a></h2></article>`
	doc, err := parser.LoadString(html)
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	item := parseGitHubTrendingItem(doc.First("article.Box-row"), time.Now())
	if item != nil {
		t.Fatalf("expected nil item, got %#v", item)
	}
}
