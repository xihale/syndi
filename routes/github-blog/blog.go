package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var githubBlogRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "The GitHub Blog",
	Example:     "github-blog",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from The GitHub Blog (native RSS, normalized)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     GitHubBlogHandler,
}

// GitHubBlogHandler handles /github-blog
func GitHubBlogHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://github.blog/feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "The GitHub Blog"
	feed.Link = "https://github.blog/"
	feed.Description = "The GitHub Blog - product updates, engineering and changelog highlights"
	return feed, nil
}
