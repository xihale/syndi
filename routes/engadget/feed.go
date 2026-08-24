package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var engadgetRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Engadget News",
	Example:     "engadget",
	Maintainers: []string{"xihale"},
	Description: "Latest tech news and reviews from Engadget (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     EngadgetHandler,
}

// EngadgetHandler handles /engadget
func EngadgetHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://www.engadget.com/rss.xml")
	if err != nil {
		return nil, err
	}
	feed.Title = "Engadget"
	feed.Link = "https://www.engadget.com/"
	feed.Description = "Breaking news from the worlds of technology and entertainment, and expert reviews of the latest consumer tech products"
	return feed, nil
}
