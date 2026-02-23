package routeutils

import "github.com/xihale/rsshub-go/pkg/models"

// AppendMappedItems maps source values to feed items and appends non-nil results.
// If limit > 0, appending stops when feed has reached the limit.
// It returns the number of appended items.
func AppendMappedItems[T any](feed *models.Feed, source []T, limit int, mapper func(T) *models.Item) int {
	if feed == nil || mapper == nil || len(source) == 0 {
		return 0
	}

	appended := 0
	for _, value := range source {
		if limit > 0 && len(feed.Items) >= limit {
			break
		}

		item := mapper(value)
		if item == nil {
			continue
		}

		feed.Items = append(feed.Items, *item)
		appended++
	}

	return appended
}
