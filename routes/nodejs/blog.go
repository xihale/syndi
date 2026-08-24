package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var nodejsBlogRoute = routeutils.RouteSpec{
	Path:        "blog",
	Name:        "Node.js Blog",
	Example:     "nodejs/blog",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from the official Node.js blog (native RSS, normalized)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     NodejsBlogHandler,
}

// NodejsBlogHandler handles /nodejs/blog
func NodejsBlogHandler(c *ctxpkg.Context) (*models.Feed, error) {
	urls := []string{
		"https://nodejs.org/en/feed/blog.xml",
		"https://nodejs.org/en/feed/rss.xml",
		"https://nodejs.org/en/blog/rss",
	}
	var feed *models.Feed
	var err error
	for _, u := range urls {
		feed, err = routeutils.GetFeed(c.Parent(), c.Client(), u)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	feed.Title = "Node.js Blog"
	feed.Link = "https://nodejs.org/en/blog"
	feed.Description = "The official Node.js Blog - news, releases and community updates"
	return feed, nil
}
