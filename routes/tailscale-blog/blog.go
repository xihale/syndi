package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var tailscaleBlogRootRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Tailscale Blog",
	Example:     "tailscale-blog",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from the Tailscale blog (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     TailscaleBlogRootHandler,
}

// TailscaleBlogRootHandler handles /tailscale-blog
func TailscaleBlogRootHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://tailscale.com/blog/index.xml")
	if err != nil {
		return nil, err
	}
	feed.Title = "Tailscale Blog"
	feed.Link = "https://tailscale.com/blog/"
	feed.Description = "Recent blog posts from Tailscale"
	return feed, nil
}
