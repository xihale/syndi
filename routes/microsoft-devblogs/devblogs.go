package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var microsoftDevBlogsRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Microsoft DevBlogs",
	Example:     "microsoft-devblogs",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from Microsoft DevBlogs (native RSS, normalized)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    time.Hour,
	Handler:     MicrosoftDevBlogsHandler,
}

// MicrosoftDevBlogsHandler handles /microsoft-devblogs
func MicrosoftDevBlogsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://devblogs.microsoft.com/feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "Microsoft DevBlogs"
	feed.Link = "https://devblogs.microsoft.com/"
	feed.Description = "Latest posts from Microsoft DevBlogs"
	return feed, nil
}
