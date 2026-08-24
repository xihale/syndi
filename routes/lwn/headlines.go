package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var lwnHeadlinesRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "LWN.net Headlines",
	Example:     "lwn",
	Maintainers: []string{"xihale"},
	Description: "Latest headlines from LWN.net (native RSS, normalized)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     LWNHeadlinesHandler,
}

// LWNHeadlinesHandler handles /lwn
func LWNHeadlinesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://lwn.net/headlines/rss")
	if err != nil {
		return nil, err
	}
	feed.Title = "LWN.net"
	feed.Link = "https://lwn.net/"
	feed.Description = "LWN.net free weekly headlines"
	return feed, nil
}
