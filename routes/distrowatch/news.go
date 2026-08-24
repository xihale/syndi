package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var distrowatchNewsRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "DistroWatch News",
	Example:     "distrowatch",
	Maintainers: []string{"xihale"},
	Description: "Latest distribution release news from DistroWatch (native RSS/RDF, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     DistroWatchNewsHandler,
}

// DistroWatchNewsHandler handles /distrowatch
func DistroWatchNewsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://distrowatch.com/news/dw.xml")
	if err != nil {
		return nil, err
	}
	feed.Title = "DistroWatch"
	feed.Link = "https://distrowatch.com/"
	feed.Description = "DistroWatch Linux/BSD distribution release news"
	return feed, nil
}
