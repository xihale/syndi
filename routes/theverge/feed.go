package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var thevergeRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "The Verge Feed",
	Example:     "theverge",
	Maintainers: []string{"xihale"},
	Description: "Latest news from The Verge (native Atom feed, normalized)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     TheVergeHandler,
}

// TheVergeHandler handles /theverge
func TheVergeHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://www.theverge.com/rss/index.xml")
	if err != nil {
		return nil, err
	}
	feed.Title = "The Verge"
	feed.Link = "https://www.theverge.com/"
	feed.Description = "The Verge - tech, culture, science and entertainment news"
	return feed, nil
}
