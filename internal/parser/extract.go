package parser

import (
	"strings"
	"time"
)

// MetaData represents extracted meta tags from HTML
type MetaData struct {
	Title       string
	Description string
	Keywords    string
	Author      string
	Canonical   string
	OGTitle     string
	OGDescription string
	OGImage     string
	OGType      string
	TwitterCard string
	Image       string
	SiteName    string
	URL         string
	PublishedTime time.Time
	ModifiedTime  time.Time
}

// ExtractMeta extracts common meta tags from the document
func (d *Document) ExtractMeta() *MetaData {
	meta := &MetaData{}

	// Basic meta tags
	meta.Title = d.Text("title")
	meta.Description, _ = d.Attr("meta[name='description']", "content")
	meta.Keywords, _ = d.Attr("meta[name='keywords']", "content")
	meta.Author, _ = d.Attr("meta[name='author']", "content")
	meta.Canonical, _ = d.Attr("link[rel='canonical']", "href")
	meta.Image, _ = d.Attr("link[rel='image_src']", "href")

	// Open Graph tags
	meta.OGTitle, _ = d.Attr("meta[property='og:title']", "content")
	if meta.OGTitle == "" {
		meta.OGTitle = meta.Title
	}
	meta.OGDescription, _ = d.Attr("meta[property='og:description']", "content")
	if meta.OGDescription == "" {
		meta.OGDescription = meta.Description
	}
	meta.OGImage, _ = d.Attr("meta[property='og:image']", "content")
	meta.OGType, _ = d.Attr("meta[property='og:type']", "content")
	meta.SiteName, _ = d.Attr("meta[property='og:site_name']", "content")
	meta.URL, _ = d.Attr("meta[property='og:url']", "content")

	// Twitter Card tags
	meta.TwitterCard, _ = d.Attr("meta[name='twitter:card']", "content")

	// Article meta
	if publishedStr, ok := d.Attr("meta[property='article:published_time']", "content"); ok {
		if t, err := time.Parse(time.RFC3339, publishedStr); err == nil {
			meta.PublishedTime = t
		}
	}
	if modifiedStr, ok := d.Attr("meta[property='article:modified_time']", "content"); ok {
		if t, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
			meta.ModifiedTime = t
		}
	}

	return meta
}

// LinkInfo represents information about a link
type LinkInfo struct {
	Href  string
	Text  string
	Title string
	Rel   string
}

// ExtractLinks extracts all links from the document
func (d *Document) ExtractLinks(selector string) []LinkInfo {
	if selector == "" {
		selector = "a"
	}

	var links []LinkInfo
	d.Each(selector, func(i int, s *Selection) {
		href, _ := s.Attr("href")
		if href == "" || strings.HasPrefix(href, "javascript:") {
			return
		}

		info := LinkInfo{
			Href: href,
			Text: strings.TrimSpace(s.Text()),
		}
		info.Title, _ = s.Attr("title")
		info.Rel, _ = s.Attr("rel")

		links = append(links, info)
	})

	return links
}

// ImageInfo represents information about an image
type ImageInfo struct {
	Src    string
	Alt    string
	Title  string
	Width  int
	Height int
}

// ExtractImages extracts all images from the document
func (d *Document) ExtractImages(selector string) []ImageInfo {
	if selector == "" {
		selector = "img"
	}

	var images []ImageInfo
	d.Each(selector, func(i int, s *Selection) {
		src, ok := s.Attr("src")
		if !ok || src == "" {
			return
		}

		info := ImageInfo{
			Src: src,
		}
		info.Alt, _ = s.Attr("alt")
		info.Title, _ = s.Attr("title")

		if widthStr, ok := s.Attr("width"); ok {
			// Parse width if needed
			_ = widthStr
		}
		if heightStr, ok := s.Attr("height"); ok {
			// Parse height if needed
			_ = heightStr
		}

		images = append(images, info)
	})

	return images
}

// ListItem represents a list item for feed generation
type ListItem struct {
	Title       string
	Link        string
	Description string
	PubDate     string
	Author      string
	Category    string
	Thumbnail   string
	Enclosure   *Enclosure
}

// Enclosure represents media enclosure (RSS)
type Enclosure struct {
	URL    string
	Type   string
	Length int64
}

// ExtractListItems extracts items from a list (common pattern for feeds)
// This is useful for blogs, news sites, etc.
func (d *Document) ExtractListItems(config ListConfig) []ListItem {
	var items []ListItem

	selector := config.ItemSelector
	if selector == "" {
		return items
	}

	d.Each(selector, func(i int, s *Selection) {
		item := ListItem{}

		// Title
		if config.TitleSelector != "" {
			if titleSel := s.Find(config.TitleSelector); titleSel != nil {
				item.Title = strings.TrimSpace(titleSel.Text())
				if item.Title == "" && config.LinkFromTitle {
					item.Link, _ = titleSel.Find("a").Attr("href")
				}
			}
		}

		// Link
		if config.LinkSelector != "" {
			if linkSel := s.Find(config.LinkSelector); linkSel != nil {
				item.Link, _ = linkSel.Attr("href")
				if item.Link == "" && config.LinkFromHref {
					item.Link, _ = linkSel.Attr("href")
				}
			}
		}

		// Description
		if config.DescriptionSelector != "" {
			if descSel := s.Find(config.DescriptionSelector); descSel != nil {
				item.Description = strings.TrimSpace(descSel.Text())
			}
		}

		// PubDate
		if config.PubDateSelector != "" {
			if dateSel := s.Find(config.PubDateSelector); dateSel != nil {
				item.PubDate = strings.TrimSpace(dateSel.Text())
			}
		}

		// Author
		if config.AuthorSelector != "" {
			if authorSel := s.Find(config.AuthorSelector); authorSel != nil {
				item.Author = strings.TrimSpace(authorSel.Text())
			}
		}

		// Category
		if config.CategorySelector != "" {
			if catSel := s.Find(config.CategorySelector); catSel != nil {
				item.Category = strings.TrimSpace(catSel.Text())
			}
		}

		// Thumbnail
		if config.ThumbnailSelector != "" {
			if thumbSel := s.Find(config.ThumbnailSelector); thumbSel != nil {
				item.Thumbnail, _ = thumbSel.Attr("src")
			}
		}

		// Only add if we have at least a title or link
		if item.Title != "" || item.Link != "" {
			items = append(items, item)
		}
	})

	return items
}

// ListConfig configures how to extract list items
type ListConfig struct {
	ItemSelector     string // Selector for each item container
	TitleSelector    string // Selector for title within item
	LinkSelector     string // Selector for link within item
	DescriptionSelector string // Selector for description
	PubDateSelector  string // Selector for publish date
	AuthorSelector   string // Selector for author
	CategorySelector string // Selector for category
	ThumbnailSelector string // Selector for thumbnail image
	LinkFromTitle    bool   // Extract link from title's href
	LinkFromHref     bool   // Extract link from href attribute
}

// ExtractJSONLD extracts JSON-LD structured data
func (d *Document) ExtractJSONLD() []map[string]interface{} {
	var results []map[string]interface{}

	d.Each("script[type='application/ld+json']", func(i int, s *Selection) {
		content := strings.TrimSpace(s.Text())
		if content == "" {
			return
		}

		// Parse JSON - would need json.Unmarshal
		// For now, skip as it requires JSON parsing
		_ = content
	})

	return results
}

// ExtractText extracts main text content using common heuristics
func (d *Document) ExtractText() string {
	// Try common main content selectors
	selectors := []string{
		"article",
		"[role='main']",
		"main",
		".content",
		".post-content",
		".entry-content",
		"#content",
		".article-body",
	}

	for _, selector := range selectors {
		if sel := d.FindSelector(selector); sel != nil && sel.Length() > 0 {
			text := strings.TrimSpace(sel.Text())
			if len(text) > 100 { // Reasonable content length
				return text
			}
		}
	}

	// Fallback to body
	return strings.TrimSpace(d.Find("body").Text())
}

// CleanText removes common noise from extracted text
func CleanText(text string) string {
	// Remove excessive whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Remove common noise patterns
	noises := []string{
		"点击这里", "点击此处",
		"Click here", "click here",
		"Read more", "read more",
		"继续阅读", "查看更多",
		"广告", "Advertisement",
	}

	for _, noise := range noises {
		text = strings.ReplaceAll(text, noise, "")
	}

	// Clean up any double spaces created
	text = strings.Join(strings.Fields(text), " ")

	return strings.TrimSpace(text)
}

// AbsoluteURL converts a relative URL to absolute
func AbsoluteURL(base, href string) string {
	if href == "" {
		return ""
	}

	// Already absolute
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	// Protocol-relative
	if strings.HasPrefix(href, "//") {
		if strings.HasPrefix(base, "https://") {
			return "https:" + href
		}
		return "http:" + href
	}

	// Root-relative
	if strings.HasPrefix(href, "/") {
		// Extract scheme and netloc
		parts := strings.Split(base, "/")
		if len(parts) >= 3 {
			return parts[0] + "//" + parts[2] + href
		}
		return base + href
	}

	// Relative to current path
	if idx := strings.LastIndex(base, "/"); idx != -1 {
		return base[:idx+1] + href
	}

	return href
}
