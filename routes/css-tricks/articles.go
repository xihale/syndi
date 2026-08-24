package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var articlesRoute = routeutils.RouteSpec{
	Path:        "articles",
	Name:        "Articles",
	Example:     "css-tricks/articles",
	Maintainers: []string{"xihale"},
	Description: "Latest articles from CSS-Tricks (native RSS, normalized)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     ArticlesHandler,
}

// ArticlesHandler handles /css-tricks/articles
func ArticlesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://css-tricks.com/feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "CSS-Tricks"
	feed.Link = "https://css-tricks.com/"
	feed.Description = "Daily articles on CSS, JavaScript, and web design from CSS-Tricks"
	return feed, nil
}
