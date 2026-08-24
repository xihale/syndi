package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var goblogRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "The Go Blog",
	Example:     "goblog",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from The Go Blog (native Atom feed, normalized)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     GoBlogHandler,
}

// GoBlogHandler handles /goblog
func GoBlogHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://go.dev/blog/feed.atom")
	if err != nil {
		return nil, err
	}
	feed.Title = "The Go Blog"
	feed.Link = "https://go.dev/blog/"
	feed.Description = "The Go Blog - official Go programming language blog"
	return feed, nil
}
