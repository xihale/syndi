package routeutils

import (
	"strings"

	"github.com/rsshub/go/pkg/models"
)

// GetAuthorString extracts display name from Item.Author
// Returns empty string if Author is nil or Name is empty
func GetAuthorString(item *models.Item) string {
	if item == nil || item.Author == nil {
		return ""
	}
	return strings.TrimSpace(item.Author.Name)
}

// SetAuthor sets item.Author field (creates Author if nil)
func SetAuthor(item *models.Item, name string, opts ...AuthorOption) {
	if item == nil {
		return
	}

	if item.Author == nil {
		item.Author = &models.Author{}
	}

	item.Author.Name = strings.TrimSpace(name)

	// Apply additional options
	for _, opt := range opts {
		opt(item.Author)
	}
}

// AuthorOption is a functional option for SetAuthor
type AuthorOption func(*models.Author)

// WithAuthorEmail sets email in author
func WithAuthorEmail(email string) AuthorOption {
	return func(a *models.Author) {
		a.Email = strings.TrimSpace(email)
	}
}

// WithAuthorURI sets URI in author
func WithAuthorURI(uri string) AuthorOption {
	return func(a *models.Author) {
		a.URI = strings.TrimSpace(uri)
	}
}

// NewAuthor creates a new Author with all fields
func NewAuthor(name, email, uri string) *models.Author {
	return &models.Author{
		Name:  strings.TrimSpace(name),
		Email: strings.TrimSpace(email),
		URI:   strings.TrimSpace(uri),
	}
}

// NormalizeAuthor normalizes author field by trimming whitespace
func NormalizeAuthor(item *models.Item) {
	if item == nil || item.Author == nil {
		return
	}

	item.Author.Name = strings.TrimSpace(item.Author.Name)
	item.Author.Email = strings.TrimSpace(item.Author.Email)
	item.Author.URI = strings.TrimSpace(item.Author.URI)

	// Remove empty author if all fields are empty
	if item.Author.Name == "" && item.Author.Email == "" && item.Author.URI == "" {
		item.Author = nil
	}
}

// GetFeedAuthorString extracts display name from Feed.Author
func GetFeedAuthorString(feed *models.Feed) string {
	if feed == nil || feed.Author == nil {
		return ""
	}
	return strings.TrimSpace(feed.Author.Name)
}

// SetFeedAuthor sets feed.Author field
func SetFeedAuthor(feed *models.Feed, name string, opts ...AuthorOption) {
	if feed == nil {
		return
	}

	if feed.Author == nil {
		feed.Author = &models.Author{}
	}

	feed.Author.Name = strings.TrimSpace(name)

	// Apply additional options
	for _, opt := range opts {
		opt(feed.Author)
	}
}

// NormalizeFeedAuthor normalizes feed author field
func NormalizeFeedAuthor(feed *models.Feed) {
	if feed == nil || feed.Author == nil {
		return
	}

	feed.Author.Name = strings.TrimSpace(feed.Author.Name)
	feed.Author.Email = strings.TrimSpace(feed.Author.Email)
	feed.Author.URI = strings.TrimSpace(feed.Author.URI)

	// Remove empty author if all fields are empty
	if feed.Author.Name == "" && feed.Author.Email == "" && feed.Author.URI == "" {
		feed.Author = nil
	}
}

// MergeAuthors combines multiple authors into a single string
func MergeAuthors(authors []string, separator string) string {
	if len(authors) == 0 {
		return ""
	}

	var result []string
	for _, author := range authors {
		author = strings.TrimSpace(author)
		if author != "" {
			result = append(result, author)
		}
	}

	return strings.Join(result, separator)
}

// ParseAuthorString parses a string containing multiple authors
func ParseAuthorString(authorStr string, separator string) []string {
	if authorStr == "" {
		return nil
	}

	parts := strings.Split(authorStr, separator)
	var authors []string
	for _, part := range parts {
		author := strings.TrimSpace(part)
		if author != "" {
			authors = append(authors, author)
		}
	}

	return authors
}

// IsValidAuthor checks if author has any non-empty fields
func IsValidAuthor(author *models.Author) bool {
	if author == nil {
		return false
	}
	return author.Name != "" || author.Email != "" || author.URI != ""
}

// FormatAuthor formats author as "Name <email> (uri)"
func FormatAuthor(author *models.Author) string {
	if author == nil {
		return ""
	}

	var parts []string

	if author.Name != "" {
		parts = append(parts, author.Name)
	}

	var withBrackets []string
	if author.Email != "" {
		withBrackets = append(withBrackets, "<"+author.Email+">")
	}
	if author.URI != "" {
		withBrackets = append(withBrackets, "("+author.URI+")")
	}

	if len(withBrackets) > 0 {
		parts = append(parts, strings.Join(withBrackets, " "))
	}

	return strings.Join(parts, " ")
}
