package test

import (
	"fmt"
	"time"

	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
)

func init() {
	cacheTTL := 10 * time.Second

	route := &models.Route{
		Path:        "/test/cache",
		Name:        "Cache Test",
		Description: "Test route to verify cache behavior (HIT/MISS)",
		Handler:     CacheTestHandler,
		Parameters:  []models.Parameter{},
		CacheTTL:    &cacheTTL,
		Categories: []models.Category{
			{Name: "Test", Description: "Test routes"},
		},
		Features: models.Features{},
	}

	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
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
