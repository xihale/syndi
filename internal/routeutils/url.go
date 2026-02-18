package routeutils

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xihale/rsshub-go/pkg/models"
)

// ResolveURL resolves a relative URL against a base URL
func ResolveURL(base, relative string) (string, error) {
	if relative == "" {
		return "", fmt.Errorf("relative URL is empty")
	}

	// If already absolute, return as-is
	if strings.HasPrefix(relative, "http://") || strings.HasPrefix(relative, "https://") {
		return relative, nil
	}

	// Handle protocol-relative URLs (//example.com/path)
	if strings.HasPrefix(relative, "//") {
		return "https:" + relative, nil
	}

	// Parse base URL
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Parse relative URL
	relURL, err := url.Parse(relative)
	if err != nil {
		return "", fmt.Errorf("invalid relative URL: %w", err)
	}

	// Resolve
	resolved := baseURL.ResolveReference(relURL)
	return resolved.String(), nil
}

// NormalizeURL removes fragments, sorts query params, and removes tracking params
func NormalizeURL(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	// Remove fragment
	parsed.Fragment = ""

	// Sort query parameters
	if parsed.RawQuery != "" {
		query := parsed.Query()
		parsed.RawQuery = query.Encode()
	}

	return parsed.String(), nil
}

// IsValidURL performs basic URL validation
func IsValidURL(u string) bool {
	if u == "" {
		return false
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	return parsed.Scheme != "" && parsed.Host != ""
}

// FormatAsAbsolute makes a relative URL absolute using base
func FormatAsAbsolute(link, base string) (string, error) {
	return ResolveURL(base, link)
}

// FormatItemsLinks updates all item links in a feed to absolute URLs
func FormatItemsLinks(feed *models.Feed) error {
	if feed == nil || feed.Items == nil {
		return nil
	}

	base := feed.Link
	if base == "" {
		return fmt.Errorf("feed link is empty, cannot resolve relative URLs")
	}

	for i := range feed.Items {
		if feed.Items[i].Link != "" {
			absolute, err := FormatAsAbsolute(feed.Items[i].Link, base)
			if err != nil {
				// Skip invalid URLs but continue processing
				continue
			}
			feed.Items[i].Link = absolute
		}
	}

	return nil
}

// BuildURL constructs a URL with query parameters
func BuildURL(base string, path string, params map[string]string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Add path
	if path != "" {
		baseURL.Path = path
	}

	// Add query parameters
	if len(params) > 0 {
		query := baseURL.Query()
		for k, v := range params {
			query.Set(k, v)
		}
		baseURL.RawQuery = query.Encode()
	}

	return baseURL.String(), nil
}

// JoinPath joins path elements to a base URL
func JoinPath(base string, parts ...string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Build path
	parts = append([]string{parsed.Path}, parts...)
	parsed.Path = strings.Join(parts, "/")

	// Clean up double slashes
	parsed.Path = strings.ReplaceAll(parsed.Path, "//", "/")

	return parsed.String(), nil
}
