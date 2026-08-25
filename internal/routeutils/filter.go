package routeutils

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/xihale/syndi/pkg/models"
)

// FilterItems keeps items matching pattern in specified fields
// If caseSensitive is false, matching is case-insensitive
func FilterItems(items []models.Item, pattern string, caseSensitive bool, fields ...string) ([]models.Item, error) {
	re, err := CompileRegex(pattern, caseSensitive)
	if err != nil {
		return nil, err
	}

	result := make([]models.Item, 0, len(items))
	for _, item := range items {
		if matchesFilterInFields(re, item, fields...) {
			result = append(result, item)
		}
	}

	return result, nil
}

// FilterOutItems removes items matching pattern
func FilterOutItems(items []models.Item, pattern string, caseSensitive bool, fields ...string) ([]models.Item, error) {
	re, err := CompileRegex(pattern, caseSensitive)
	if err != nil {
		return nil, err
	}

	result := make([]models.Item, 0, len(items))
	for _, item := range items {
		if !matchesFilterInFields(re, item, fields...) {
			result = append(result, item)
		}
	}

	return result, nil
}

// FilterByTime keeps items with pubDate within last N seconds
func FilterByTime(items []models.Item, seconds int64) []models.Item {
	cutoff := time.Now().Add(-time.Duration(seconds) * time.Second)

	result := make([]models.Item, 0, len(items))
	for _, item := range items {
		if !item.PubDate.IsZero() && item.PubDate.After(cutoff) {
			result = append(result, item)
		}
	}

	return result
}

// SortByPubDate sorts items by pubDate in place — the caller's backing
// array is modified, so cached/shared slices must be copied first.
func SortByPubDate(items []models.Item, descending bool) {
	sort.Slice(items, func(i, j int) bool {
		// Handle zero dates
		if items[i].PubDate.IsZero() {
			return !descending
		}
		if items[j].PubDate.IsZero() {
			return descending
		}

		if descending {
			return items[i].PubDate.After(items[j].PubDate)
		}
		return items[i].PubDate.Before(items[j].PubDate)
	})
}

// ApplyLimit truncates to max items
func ApplyLimit(items []models.Item, limit int) []models.Item {
	if limit <= 0 || limit > len(items) {
		return items
	}
	return items[:limit]
}

// ApplyParameters applies query parameters to filter items
// Supports: limit, filter, filterout, filter_time, sorted
func ApplyParameters(items []models.Item, query url.Values) ([]models.Item, error) {
	result := items

	// 1. Filter by time window
	if filterTime := query.Get("filter_time"); filterTime != "" {
		var seconds int64
		_, err := fmt.Sscanf(filterTime, "%d", &seconds)
		if err == nil && seconds > 0 {
			result = FilterByTime(result, seconds)
		}
	}

	// 2. Filter items (include)
	if filter := query.Get("filter"); filter != "" {
		var err error
		result, err = FilterItems(result, filter, false, "title", "description", "author", "category")
		if err != nil {
			return result, fmt.Errorf("invalid filter regex %q: %w", filter, err)
		}
	}

	// 3. Filter out items (exclude)
	if filterOut := query.Get("filterout"); filterOut != "" {
		var err error
		result, err = FilterOutItems(result, filterOut, false, "title", "description", "author", "category")
		if err != nil {
			return result, fmt.Errorf("invalid filterout regex %q: %w", filterOut, err)
		}
	}

	// 4. Sort items
	sorted := true // Default to sorted
	if sortedStr := query.Get("sorted"); sortedStr != "" {
		sorted = strings.ToLower(sortedStr) == "true"
	}
	if sorted {
		SortByPubDate(result, true)
	}

	// 5. Limit items
	if limit := query.Get("limit"); limit != "" {
		var l int
		_, err := fmt.Sscanf(limit, "%d", &l)
		if err == nil && l > 0 {
			result = ApplyLimit(result, l)
		}
	}

	return result, nil
}

// CompileRegex compiles a regex pattern with proper flags
func CompileRegex(pattern string, caseSensitive bool) (*regexp.Regexp, error) {
	if !caseSensitive {
		// Add (?i) flag for case-insensitive matching
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// matchesFilterInFields checks if an item matches in specific fields only
func matchesFilterInFields(re *regexp.Regexp, item models.Item, fields ...string) bool {
	for _, field := range fields {
		switch field {
		case "title":
			if re.MatchString(item.Title) {
				return true
			}
		case "description":
			if re.MatchString(item.Description) {
				return true
			}
		case "author":
			if item.Author != nil && re.MatchString(item.Author.Name) {
				return true
			}
		case "category":
			for _, category := range item.Categories {
				if re.MatchString(category) {
					return true
				}
			}
		}
	}
	return false
}

// UniqueBy removes duplicate items based on a key function
func UniqueBy(items []models.Item, keyFn func(item models.Item) string) []models.Item {
	seen := make(map[string]bool)
	result := make([]models.Item, 0, len(items))

	for _, item := range items {
		key := keyFn(item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result
}

// UniqueByLink removes duplicate items based on link
func UniqueByLink(items []models.Item) []models.Item {
	return UniqueBy(items, func(item models.Item) string { return item.Link })
}

// UniqueByGUID removes duplicate items based on GUID
func UniqueByGUID(items []models.Item) []models.Item {
	return UniqueBy(items, func(item models.Item) string { return item.GUID })
}
