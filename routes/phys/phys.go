package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var physRootRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Phys.org News",
	Example:     "phys",
	Maintainers: []string{"xihale"},
	Description: "Latest science and technology news from Phys.org (native RSS, normalized)",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     PhysRootHandler,
}

// PhysRootHandler handles /phys
func PhysRootHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://phys.org/rss-feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "Phys.org"
	feed.Link = "https://phys.org/"
	feed.Description = "Science and technology news from Phys.org"
	return feed, nil
}
