package test

import (
	"fmt"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var cacheTestRoute = routeutils.RouteSpec{
	Path:        "/test/cache",
	Name:        "Cache Test",
	Example:     "test/cache",
	Maintainers: []string{"xihale"},
	Description: "Test route to verify cache behavior (HIT/MISS)",
	Categories: []models.Category{
		{Name: "Test", Description: "Test routes"},
	},
	Features: models.Features{},
	CacheTTL: 10 * time.Second,
	Handler:  CacheTestHandler,
}

// CacheTestHandler generates a test feed with timestamp to demonstrate caching
func CacheTestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	now := time.Now()

	feed := &models.Feed{
		Title:       "Cache Test Feed",
		Link:        "https://example.com/test/cache",
		Description: "This feed demonstrates caching behavior. The timestamp updates on every cache miss.",
		Items: []models.Item{
			{
				Title:       fmt.Sprintf("Cache Test Item - Generated at %s", now.Format(time.RFC3339)),
				Link:        "https://example.com/test/item",
				GUID:        fmt.Sprintf("item-%d", now.Unix()),
				PubDate:     now,
				Description: fmt.Sprintf("This item was generated at %s. If you see the same timestamp after refreshing, the cache is working.", now.Format(time.RFC3339)),
			},
		},
	}

	return feed, nil
}
