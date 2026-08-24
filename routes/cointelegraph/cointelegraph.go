package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var cointelegraphRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Cointelegraph",
	Example:     "cointelegraph",
	Maintainers: []string{"xihale"},
	Description: "Latest crypto and blockchain news from Cointelegraph (native RSS, normalized)",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     CointelegraphHandler,
}

// CointelegraphHandler handles /cointelegraph
func CointelegraphHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://cointelegraph.com/rss")
	if err != nil {
		return nil, err
	}
	feed.Title = "Cointelegraph"
	feed.Link = "https://cointelegraph.com/"
	feed.Description = "Latest crypto and blockchain news from Cointelegraph"
	return feed, nil
}
