package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var solidotRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Solidot News",
	Example:     "solidot",
	Maintainers: []string{"xihale"},
	Description: "Latest news from Solidot (native RSS, normalized)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     SolidotHandler,
}

// SolidotHandler handles /solidot
func SolidotHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://www.solidot.org/index.rss")
	if err != nil {
		return nil, err
	}
	feed.Title = "Solidot"
	feed.Link = "https://www.solidot.org/"
	feed.Description = "Solidot 奇客的最新资讯"
	return feed, nil
}
