package parser

import (
	"strings"
	"testing"
)

func TestExtractMeta(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<meta name="description" content="Test description">
	<meta name="keywords" content="test, keywords">
	<meta name="author" content="Test Author">
	<link rel="canonical" href="https://example.com/page">
	<meta property="og:title" content="OG Title">
	<meta property="og:description" content="OG Description">
	<meta property="og:image" content="https://example.com/image.jpg">
	<meta property="og:type" content="website">
	<meta property="og:site_name" content="Test Site">
	<meta property="og:url" content="https://example.com/page">
	<meta name="twitter:card" content="summary_large_image">
	<meta property="article:published_time" content="2024-01-15T10:00:00Z">
	<meta property="article:modified_time" content="2024-01-16T12:00:00Z">
</head>
<body>
</body>
</html>`

	doc, _ := LoadString(html)
	meta := doc.ExtractMeta()

	if meta.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got '%s'", meta.Title)
	}

	if meta.Description != "Test description" {
		t.Errorf("expected description 'Test description', got '%s'", meta.Description)
	}

	if meta.Keywords != "test, keywords" {
		t.Errorf("expected keywords 'test, keywords', got '%s'", meta.Keywords)
	}

	if meta.Author != "Test Author" {
		t.Errorf("expected author 'Test Author', got '%s'", meta.Author)
	}

	if meta.Canonical != "https://example.com/page" {
		t.Errorf("expected canonical 'https://example.com/page', got '%s'", meta.Canonical)
	}

	if meta.OGTitle != "OG Title" {
		t.Errorf("expected OG title 'OG Title', got '%s'", meta.OGTitle)
	}

	if meta.OGDescription != "OG Description" {
		t.Errorf("expected OG description 'OG Description', got '%s'", meta.OGDescription)
	}

	if meta.OGImage != "https://example.com/image.jpg" {
		t.Errorf("expected OG image 'https://example.com/image.jpg', got '%s'", meta.OGImage)
	}

	if meta.SiteName != "Test Site" {
		t.Errorf("expected site name 'Test Site', got '%s'", meta.SiteName)
	}

	if meta.PublishedTime.IsZero() {
		t.Error("expected published time to be set")
	}

	if meta.ModifiedTime.IsZero() {
		t.Error("expected modified time to be set")
	}
}

func TestExtractMeta_Fallback(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Page Title</title>
	<meta name="description" content="Description">
</head>
<body>
</body>
</html>`

	doc, _ := LoadString(html)
	meta := doc.ExtractMeta()

	// OG should fall back to basic meta
	if meta.OGTitle != "Page Title" {
		t.Errorf("expected OG title to fallback to 'Page Title', got '%s'", meta.OGTitle)
	}

	if meta.OGDescription != "Description" {
		t.Errorf("expected OG description to fallback to 'Description', got '%s'", meta.OGDescription)
	}
}

func TestExtractLinks(t *testing.T) {
	html := `<html><body>
		<a href="/link1">Link 1</a>
		<a href="https://example.com/link2" title="Link 2 Title">Link 2</a>
		<a href="javascript:void(0)">JavaScript Link</a>
		<a>No href</a>
	</body></html>`

	doc, _ := LoadString(html)
	links := doc.ExtractLinks("")

	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}

	if links[0].Href != "/link1" {
		t.Errorf("expected href '/link1', got '%s'", links[0].Href)
	}

	if links[0].Text != "Link 1" {
		t.Errorf("expected text 'Link 1', got '%s'", links[0].Text)
	}

	if links[1].Href != "https://example.com/link2" {
		t.Errorf("expected href 'https://example.com/link2', got '%s'", links[1].Href)
	}

	if links[1].Title != "Link 2 Title" {
		t.Errorf("expected title 'Link 2 Title', got '%s'", links[1].Title)
	}
}

func TestExtractImages(t *testing.T) {
	html := `<html><body>
		<img src="/image1.jpg" alt="Image 1" title="Title 1">
		<img src="https://example.com/image2.jpg" alt="Image 2" width="100" height="200">
		<img alt="No src">
	</body></html>`

	doc, _ := LoadString(html)
	images := doc.ExtractImages("")

	if len(images) != 2 {
		t.Errorf("expected 2 images, got %d", len(images))
	}

	if images[0].Src != "/image1.jpg" {
		t.Errorf("expected src '/image1.jpg', got '%s'", images[0].Src)
	}

	if images[0].Alt != "Image 1" {
		t.Errorf("expected alt 'Image 1', got '%s'", images[0].Alt)
	}

	if images[1].Src != "https://example.com/image2.jpg" {
		t.Errorf("expected src 'https://example.com/image2.jpg', got '%s'", images[1].Src)
	}
}

func TestExtractListItems(t *testing.T) {
	html := `<html><body>
		<div class="article-list">
			<div class="article-item">
				<h2><a href="/article1">Article 1</a></h2>
				<p class="summary">Summary 1</p>
				<span class="date">2024-01-01</span>
				<span class="author">Author 1</span>
			</div>
			<div class="article-item">
				<h2><a href="/article2">Article 2</a></h2>
				<p class="summary">Summary 2</p>
				<span class="date">2024-01-02</span>
				<span class="author">Author 2</span>
			</div>
		</div>
	</body></html>`

	doc, _ := LoadString(html)
	config := ListConfig{
		ItemSelector:     ".article-item",
		TitleSelector:    "h2 a",
		LinkSelector:     "h2 a",
		LinkFromHref:     true,
		DescriptionSelector: ".summary",
		PubDateSelector:  ".date",
		AuthorSelector:   ".author",
	}

	items := doc.ExtractListItems(config)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].Title != "Article 1" {
		t.Errorf("expected title 'Article 1', got '%s'", items[0].Title)
	}

	if items[0].Link != "/article1" {
		t.Errorf("expected link '/article1', got '%s'", items[0].Link)
	}

	if items[0].Description != "Summary 1" {
		t.Errorf("expected description 'Summary 1', got '%s'", items[0].Description)
	}

	if items[0].PubDate != "2024-01-01" {
		t.Errorf("expected pubdate '2024-01-01', got '%s'", items[0].PubDate)
	}

	if items[0].Author != "Author 1" {
		t.Errorf("expected author 'Author 1', got '%s'", items[0].Author)
	}
}

func TestExtractListItems_WithTitleFromLink(t *testing.T) {
	html := `<html><body>
		<div class="item">
			<a href="/link1">Link Title 1</a>
		</div>
	</body></html>`

	doc, _ := LoadString(html)
	config := ListConfig{
		ItemSelector:  ".item",
		TitleSelector: "a",
		LinkSelector:  "a",
		LinkFromTitle: true,
	}

	items := doc.ExtractListItems(config)

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Title != "Link Title 1" {
		t.Errorf("expected title 'Link Title 1', got '%s'", items[0].Title)
	}

	if items[0].Link != "/link1" {
		t.Errorf("expected link '/link1', got '%s'", items[0].Link)
	}
}

func TestExtractListItems_SkipInvalid(t *testing.T) {
	html := `<html><body>
		<div class="item">
			<span>No title or link</span>
		</div>
		<div class="item">
			<h2>Has Title</h2>
		</div>
		<div class="item">
			<a href="/link">Has Link</a>
		</div>
	</body></html>`

	doc, _ := LoadString(html)
	config := ListConfig{
		ItemSelector:  ".item",
		TitleSelector: "h2",
		LinkSelector:  "a",
		LinkFromHref:  true,
	}

	items := doc.ExtractListItems(config)

	if len(items) != 2 {
		t.Errorf("expected 2 items (skipping first), got %d", len(items))
	}
}

func TestExtractText(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<article>
		<p>Main content here</p>
		<p>More content</p>
	</article>
	<div class="sidebar">Sidebar content</div>
</body>
</html>`

	doc, _ := LoadString(html)
	text := doc.ExtractText()

	if !strings.Contains(text, "Main content here") {
		t.Error("expected text to contain main content")
	}
}

func TestExtractText_Fallback(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<p>Body content</p>
</body>
</html>`

	doc, _ := LoadString(html)
	text := doc.ExtractText()

	if !strings.Contains(text, "Body content") {
		t.Error("expected text to contain body content")
	}
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove extra whitespace",
			input:    "Hello    World   Test",
			expected: "Hello World Test",
		},
		{
			name:     "remove click noise",
			input:    "Content click here for more details",
			expected: "Content for more details",
		},
		{
			name:     "remove Chinese click here",
			input:    "Content 点击这里 for details",
			expected: "Content for details",
		},
		{
			name:     "trim spaces",
			input:    "   Hello World   ",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanText(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestAbsoluteURL(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		href     string
		expected string
	}{
		{
			name:     "already absolute http",
			base:     "https://example.com/page",
			href:     "http://other.com/link",
			expected: "http://other.com/link",
		},
		{
			name:     "already absolute https",
			base:     "https://example.com/page",
			href:     "https://other.com/link",
			expected: "https://other.com/link",
		},
		{
			name:     "protocol relative",
			base:     "https://example.com/page",
			href:     "//cdn.example.com/file.js",
			expected: "https://cdn.example.com/file.js",
		},
		{
			name:     "root relative",
			base:     "https://example.com/path/to/page",
			href:     "/absolute/path",
			expected: "https://example.com/absolute/path",
		},
		{
			name:     "relative path",
			base:     "https://example.com/path/to/page",
			href:     "other.html",
			expected: "https://example.com/path/to/other.html",
		},
		{
			name:     "relative with directory",
			base:     "https://example.com/path/to/",
			href:     "other.html",
			expected: "https://example.com/path/to/other.html",
		},
		{
			name:     "empty href",
			base:     "https://example.com",
			href:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AbsoluteURL(tt.base, tt.href)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
