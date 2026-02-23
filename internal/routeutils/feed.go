package routeutils

import (
	"time"

	"github.com/xihale/rsshub-go/pkg/models"
)

// FeedOptions contains optional fields for NewFeedWithOptions
type FeedOptions struct {
	Title       string
	Link        string
	Description string
	Image       string
	Author      *models.Author
	Language    string
	Updated     *time.Time
}

// NewFeed creates a Feed with required fields
func NewFeed(title, link, description string) *models.Feed {
	return &models.Feed{
		Title:       title,
		Link:        link,
		Description: description,
		Items:       make([]models.Item, 0),
	}
}

// NewFeedWithOptions creates Feed with optional fields
func NewFeedWithOptions(opts FeedOptions) *models.Feed {
	feed := &models.Feed{
		Title:       opts.Title,
		Link:        opts.Link,
		Description: opts.Description,
		Items:       make([]models.Item, 0),
		Author:      opts.Author,
		Updated:     opts.Updated,
	}
	return feed
}

// ItemOptions contains optional fields for NewItemWithOptions
type ItemOptions struct {
	Title       string
	Link        string
	Description string
	PubDate     time.Time
	GUID        string
	Author      *models.Author
	Categories  []string
	Updated     *time.Time
	Content     *ItemContent
}

// ItemContent represents structured content with HTML and text versions
type ItemContent struct {
	HTML string
	Text string
}

// NewItem creates an Item with required fields
func NewItem(title, link, description string, pubDate time.Time) *models.Item {
	item := &models.Item{
		Title:       title,
		Link:        link,
		Description: description,
		PubDate:     pubDate,
	}
	applyItemDefaults(item)
	return item
}

// NewItemWithOptions creates Item with all optional fields
func NewItemWithOptions(opts ItemOptions) *models.Item {
	item := &models.Item{
		Title:       opts.Title,
		Link:        opts.Link,
		Description: opts.Description,
		PubDate:     opts.PubDate,
		GUID:        opts.GUID,
		Author:      opts.Author,
		Categories:  opts.Categories,
		Updated:     opts.Updated,
	}

	// Set content if provided
	if opts.Content != nil {
		// Note: This will require adding Content field to models.Item
		// For now, we can store HTML in Description
		if opts.Content.HTML != "" {
			item.Description = opts.Content.HTML
		}
	}

	applyItemDefaults(item)

	return item
}

// AddItem safely appends item to feed (creates Items slice if nil)
func AddItem(feed *models.Feed, item *models.Item) {
	if feed == nil || item == nil {
		return
	}
	if feed.Items == nil {
		feed.Items = make([]models.Item, 0, 10)
	}
	applyItemDefaults(item)
	feed.Items = append(feed.Items, *item)
}

// AddItems appends multiple items to feed
func AddItems(feed *models.Feed, items ...*models.Item) {
	if feed == nil || len(items) == 0 {
		return
	}
	if feed.Items == nil {
		feed.Items = make([]models.Item, 0, len(items))
	}
	for _, item := range items {
		if item != nil {
			applyItemDefaults(item)
			feed.Items = append(feed.Items, *item)
		}
	}
}

func applyItemDefaults(item *models.Item) {
	if item == nil {
		return
	}
	if item.GUID == "" && item.Link != "" {
		item.GUID = item.Link
	}
}


// SetCategories ensures item.Categories is non-nil and adds categories
func SetCategories(item *models.Item, categories ...string) {
	if item == nil {
		return
	}
	if item.Categories == nil {
		item.Categories = make([]string, 0, len(categories))
	}
	for _, cat := range categories {
		if cat != "" {
			item.Categories = append(item.Categories, cat)
		}
	}
}

// SetContent sets content.html and content.text
// Note: This requires Content field to be added to models.Item
func SetContent(item *models.Item, html, text string) {
	if item == nil {
		return
	}
	// For now, store HTML in Description
	// When Content field is added to models.Item, update this
	if html != "" {
		item.Description = html
	}
}

// SetUpdated sets item.Updated from pubDate if not already set
func SetUpdated(item *models.Item, t time.Time) {
	if item == nil {
		return
	}
	if item.Updated == nil && !t.IsZero() {
		item.Updated = &t
	}
}

// SetItemAuthor sets item.Author
func SetItemAuthor(item *models.Item, name, email, uri string) {
	if item == nil {
		return
	}
	item.Author = &models.Author{
		Name:  name,
		Email: email,
		URI:   uri,
	}
}

// MergeItems combines items from multiple feeds
func MergeItems(feeds ...*models.Feed) []models.Item {
	result := make([]models.Item, 0)
	for _, feed := range feeds {
		if feed != nil && len(feed.Items) > 0 {
			result = append(result, feed.Items...)
		}
	}
	return result
}

// CloneFeed creates a shallow copy of a feed
func CloneFeed(feed *models.Feed) *models.Feed {
	if feed == nil {
		return nil
	}
	return &models.Feed{
		Title:       feed.Title,
		Description: feed.Description,
		Link:        feed.Link,
		Items:       append([]models.Item{}, feed.Items...),
		Author:      feed.Author,
		Updated:     feed.Updated,
	}
}

// GetItemCount returns the number of items in a feed (safe for nil feeds)
func GetItemCount(feed *models.Feed) int {
	if feed == nil {
		return 0
	}
	return len(feed.Items)
}
