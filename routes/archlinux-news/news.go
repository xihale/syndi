package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var archlinuxNewsRootRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Arch Linux News",
	Example:     "archlinux-news",
	Maintainers: []string{"xihale"},
	Description: "Latest news from archlinux.org (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     ArchlinuxNewsRootHandler,
}

// ArchlinuxNewsRootHandler handles /archlinux-news
func ArchlinuxNewsRootHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://archlinux.org/feeds/news/")
	if err != nil {
		return nil, err
	}
	feed.Title = "Arch Linux News"
	feed.Link = "https://archlinux.org/news/"
	feed.Description = "Latest news from the Arch Linux project"
	return feed, nil
}
