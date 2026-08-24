package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var techcrunchRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "TechCrunch Feed",
	Example:     "techcrunch",
	Maintainers: []string{"xihale"},
	Description: "Latest startup and technology news from TechCrunch (native RSS, normalized)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     TechCrunchHandler,
}

// TechCrunchHandler handles /techcrunch
func TechCrunchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://techcrunch.com/feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "TechCrunch"
	feed.Link = "https://techcrunch.com/"
	feed.Description = "TechCrunch - startup and technology news"
	return feed, nil
}
