package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var rustBlogRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "The Rust Blog",
	Example:     "rustblog",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from the official Rust language blog (native Atom feed, normalized)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     RustBlogHandler,
}

// RustBlogHandler handles /rustblog
func RustBlogHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://blog.rust-lang.org/feed.xml")
	if err != nil {
		return nil, err
	}
	feed.Title = "The Rust Blog"
	feed.Link = "https://blog.rust-lang.org/"
	feed.Description = "The Rust Blog - official Rust programming language blog"
	return feed, nil
}
