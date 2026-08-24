package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var itsfossRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "It's FOSS Feed",
	Example:     "itsfoss",
	Maintainers: []string{"xihale"},
	Description: "Latest Linux news and tutorials from It's FOSS (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     ItsFossHandler,
}

// ItsFossHandler handles /itsfoss
func ItsFossHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://itsfoss.com/rss/")
	if err != nil {
		return nil, err
	}
	feed.Title = "It's FOSS"
	feed.Link = "https://itsfoss.com/"
	feed.Description = "It's FOSS - open-source news, Linux tutorials and tips"
	return feed, nil
}
