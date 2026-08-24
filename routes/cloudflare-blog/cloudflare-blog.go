package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var cloudflareBlogRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Cloudflare Blog",
	Example:     "cloudflare-blog",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from the Cloudflare blog (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    time.Hour,
	Handler:     CloudflareBlogHandler,
}

// CloudflareBlogHandler handles /cloudflare-blog
func CloudflareBlogHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://blog.cloudflare.com/rss/")
	if err != nil {
		return nil, err
	}
	feed.Title = "Cloudflare Blog"
	feed.Link = "https://blog.cloudflare.com/"
	feed.Description = "Latest posts from the Cloudflare blog"
	return feed, nil
}
