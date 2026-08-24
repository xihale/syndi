package routeutils

import (
	"testing"
	"time"

	"github.com/xihale/syndi/pkg/models"
)

func TestAppendMappedItems(t *testing.T) {
	type source struct {
		id    int
		valid bool
	}

	feed := NewFeed("t", "https://example.com", "d")
	src := []source{
		{id: 1, valid: true},
		{id: 2, valid: false},
		{id: 3, valid: true},
	}

	got := AppendMappedItems(feed, src, 0, func(v source) *models.Item {
		if !v.valid {
			return nil
		}
		return &models.Item{
			Title:   "item",
			Link:    "https://example.com/item",
			GUID:    "id",
			PubDate: time.Now(),
		}
	})

	if got != 2 {
		t.Fatalf("AppendMappedItems() appended = %d, want 2", got)
	}
	if len(feed.Items) != 2 {
		t.Fatalf("AppendMappedItems() feed length = %d, want 2", len(feed.Items))
	}
}

func TestAppendMappedItemsLimit(t *testing.T) {
	feed := NewFeed("t", "https://example.com", "d")
	src := []int{1, 2, 3, 4}

	got := AppendMappedItems(feed, src, 2, func(v int) *models.Item {
		return &models.Item{
			Title:   "item",
			Link:    "https://example.com/item",
			GUID:    "id",
			PubDate: time.Now(),
		}
	})

	if got != 2 {
		t.Fatalf("AppendMappedItems() appended = %d, want 2", got)
	}
	if len(feed.Items) != 2 {
		t.Fatalf("AppendMappedItems() feed length = %d, want 2", len(feed.Items))
	}
}
