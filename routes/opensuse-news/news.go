package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var openSUSENewsRootRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "openSUSE News",
	Example:     "opensuse-news",
	Maintainers: []string{"xihale"},
	Description: "Latest news from the openSUSE project (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     OpenSUSENewsRootHandler,
}

// OpenSUSENewsRootHandler handles /opensuse-news
func OpenSUSENewsRootHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://news.opensuse.org/feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "openSUSE News"
	feed.Link = "https://news.opensuse.org/"
	feed.Description = "Latest news from the openSUSE project"
	return feed, nil
}
