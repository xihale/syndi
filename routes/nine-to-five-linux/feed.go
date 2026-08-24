package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var nineToFiveLinuxRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "9to5Linux Feed",
	Example:     "nine-to-five-linux",
	Maintainers: []string{"xihale"},
	Description: "Latest Linux news from 9to5Linux (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     NineToFiveLinuxHandler,
}

// NineToFiveLinuxHandler handles /nine-to-five-linux
func NineToFiveLinuxHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://9to5linux.com/feed")
	if err != nil {
		return nil, err
	}
	feed.Title = "9to5Linux"
	feed.Link = "https://9to5linux.com/"
	feed.Description = "9to5Linux - Linux open-source news weekly"
	return feed, nil
}
