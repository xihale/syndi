package middleware

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xihale/syndi/pkg/models"
)

const (
	// Query parameter keys
	paramLimit      = "limit"
	paramFilter     = "filter"
	paramFilterOut  = "filterout"
	paramFilterTime = "filter_time"
	paramSorted     = "sorted"
	paramBrief      = "brief"

	// Context keys for feed storage
	contextFeedKey = "_rsshub_feed"
)

// Parameter returns a middleware that processes query parameters to modify feed items
func Parameter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only process if a feed was generated
		feed, exists := c.Get(contextFeedKey)
		if !exists {
			return
		}

		feedObj, ok := feed.(*models.Feed)
		if !ok || feedObj == nil {
			return
		}

		// Process query parameters in order
		feedObj.Items = ProcessFeed(c, feedObj.Items)
	}
}

// ProcessFeed applies all query parameter transformations to a feed's items
// This function can be called directly by caching handlers
func ProcessFeed(c *gin.Context, items []models.Item) []models.Item {
	// 1. Filter by time window
	if filterTime := c.Query(paramFilterTime); filterTime != "" {
		items = filterByTime(items, filterTime)
	}

	// 2. Filter items (include)
	if filter := c.Query(paramFilter); filter != "" {
		items = filterItems(items, filter, false)
	}

	// 3. Filter out items (exclude)
	if filterOut := c.Query(paramFilterOut); filterOut != "" {
		items = filterItems(items, filterOut, true)
	}

	// 4. Sort items
	sorted := true // Default to sorted
	if sortedStr := c.Query(paramSorted); sortedStr != "" {
		sorted = strings.ToLower(sortedStr) == "true"
	}
	if sorted {
		items = sortItemsByPubDate(items)
	}

	// 5. Limit items
	if limit := c.Query(paramLimit); limit != "" {
		items = limitItems(items, limit)
	}

	// 6. Brief mode (truncate descriptions)
	if brief := c.Query(paramBrief); brief != "" {
		items = applyBriefMode(items, brief)
	}

	return items
}

// filterItems filters items based on regex pattern
// include=true keeps matching items, include=false removes matching items
func filterItems(items []models.Item, pattern string, exclude bool) []models.Item {
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Invalid regex, return all items
		return items
	}

	result := make([]models.Item, 0, len(items))

	for _, item := range items {
		matches := matchesFilter(re, item)

		if exclude {
			// filterout: exclude matching items
			if !matches {
				result = append(result, item)
			}
		} else {
			// filter: include only matching items
			if matches {
				result = append(result, item)
			}
		}
	}

	return result
}

// matchesFilter checks if an item matches the filter pattern
func matchesFilter(re *regexp.Regexp, item models.Item) bool {
	// Check title
	if re.MatchString(item.Title) {
		return true
	}

	// Check description
	if re.MatchString(item.Description) {
		return true
	}

	// Check author
	if item.Author != nil && re.MatchString(item.Author.Name) {
		return true
	}

	// Check categories
	for _, category := range item.Categories {
		if re.MatchString(category) {
			return true
		}
	}

	return false
}

// filterByTime filters items to only include those within a time window
func filterByTime(items []models.Item, filterTime string) []models.Item {
	// Parse filter_time as seconds
	var seconds int64
	_, err := fmt.Sscanf(filterTime, "%d", &seconds)
	if err != nil {
		return items
	}

	cutoff := time.Now().Add(-time.Duration(seconds) * time.Second)

	result := make([]models.Item, 0, len(items))
	for _, item := range items {
		// Check if item is within time window
		if !item.PubDate.IsZero() && item.PubDate.After(cutoff) {
			result = append(result, item)
		}
	}

	return result
}

// limitItems limits the number of items returned
func limitItems(items []models.Item, limitStr string) []models.Item {
	var limit int
	_, err := fmt.Sscanf(limitStr, "%d", &limit)
	if err != nil || limit <= 0 {
		return items
	}

	if limit > len(items) {
		return items
	}

	return items[:limit]
}

// applyBriefMode truncates item descriptions to a specified length
func applyBriefMode(items []models.Item, briefStr string) []models.Item {
	var maxLength int
	_, err := fmt.Sscanf(briefStr, "%d", &maxLength)
	if err != nil || maxLength <= 0 {
		// Default to 100 characters if parsing fails
		maxLength = 100
	}

	for i := range items {
		if len(items[i].Description) > maxLength {
			items[i].Description = items[i].Description[:maxLength] + "..."
		}
	}

	return items
}

// sortItemsByPubDate sorts items by publication date (newest first)
func sortItemsByPubDate(items []models.Item) []models.Item {
	// Create a copy to avoid modifying the original
	sorted := make([]models.Item, len(items))
	copy(sorted, items)

	sort.Slice(sorted, func(i, j int) bool {
		// Handle zero dates
		if sorted[i].PubDate.IsZero() {
			return false
		}
		if sorted[j].PubDate.IsZero() {
			return true
		}

		// Sort by descending date (newest first)
		return sorted[i].PubDate.After(sorted[j].PubDate)
	})

	return sorted
}
