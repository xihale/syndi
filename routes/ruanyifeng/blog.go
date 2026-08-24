package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var ruanyifengRootRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "阮一峰的网络日志",
	Example:     "ruanyifeng",
	Maintainers: []string{"xihale"},
	Description: "阮一峰（Ruan YiFeng）博客最新文章 (native Atom, normalized)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     RuanyifengRootHandler,
}

// RuanyifengRootHandler handles /ruanyifeng
func RuanyifengRootHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://www.ruanyifeng.com/blog/atom.xml")
	if err != nil {
		return nil, err
	}
	feed.Title = "阮一峰的网络日志"
	feed.Link = "https://www.ruanyifeng.com/blog/"
	feed.Description = "阮一峰（Ruan YiFeng）的博客最新文章"
	return feed, nil
}
