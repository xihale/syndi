package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var sspaiRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "sspai Feed",
	Example:     "sspai",
	Maintainers: []string{"xihale"},
	Description: "Latest articles from 少数派 sspai (native RSS, normalized)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     SspaiHandler,
}

// SspaiHandler handles /sspai
func SspaiHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://sspai.com/feed")
	if err != nil {
		return nil, err
	}
	feed.Title = "少数派"
	feed.Link = "https://sspai.com/"
	feed.Description = "少数派 - 高效工作，品质生活"
	return feed, nil
}
