package main

import (
	"time"

	ctxpkg "github.com/rsshub/go/pkg/context"
	"github.com/rsshub/go/pkg/models"
	"github.com/rsshub/go/pkg/registry"
)

func init() {
	route := &models.Route{
		Path:         "/example",
		Name:         "Example Feed",
		Example:      "/example",
		Maintainers:  []string{"yourname"},
		Description:  "An example RSS feed",
		Categories:   []models.Category{{Name: "other"}},
		Features:     models.Features{},
		Handler:      ExampleHandler,
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// ExampleHandler is an example route handler
func ExampleHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return &models.Feed{
		Title:       "Example Feed",
		Link:        c.BaseURL() + "/example",
		Description: "An example RSS feed",
		Items: []models.Item{
			{
				Title:       "Example Item 1",
				Link:        c.BaseURL() + "/example/1",
				GUID:        "example-1",
				Description: "This is an example item",
				PubDate:     time.Now(),
			},
		},
	}, nil
}
