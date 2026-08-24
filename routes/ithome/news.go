package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var ithomeRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "ITHome News",
	Example:     "ithome",
	Maintainers: []string{"xihale"},
	Description: "Latest tech news from ITHome (native RSS, normalized)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     ITHomeHandler,
}

// ITHomeHandler handles /ithome
func ITHomeHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://www.ithome.com/rss/")
	if err != nil {
		return nil, err
	}
	feed.Title = "IT之家"
	feed.Link = "https://www.ithome.com/"
	feed.Description = "IT之家最新科技资讯"
	return feed, nil
}
