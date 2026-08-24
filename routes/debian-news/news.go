package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var debianNewsRootRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Debian News",
	Example:     "debian-news",
	Maintainers: []string{"xihale"},
	Description: "Latest news from the Debian project (the News page serves RDF natively)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     DebianNewsRootHandler,
}

// DebianNewsRootHandler handles /debian-news
func DebianNewsRootHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://www.debian.org/News/news")
	if err != nil {
		return nil, err
	}
	feed.Title = "Debian News"
	feed.Link = "https://www.debian.org/News/"
	feed.Description = "Latest news from the Debian project"
	return feed, nil
}
