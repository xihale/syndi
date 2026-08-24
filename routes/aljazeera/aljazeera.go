package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const aljazeeraFeedURL = "https://www.aljazeera.com/xml/rss/all.xml"

var aljazeeraRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Al Jazeera All News",
	Example:     "aljazeera",
	Maintainers: []string{"xihale"},
	Description: "Al Jazeera English latest news from the official RSS feed",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    15 * time.Minute,
	Handler:     AlJazeeraHandler,
}

// AlJazeeraHandler handles /aljazeera
func AlJazeeraHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), aljazeeraFeedURL)
	if err != nil {
		return nil, err
	}
	if feed.Title == "" {
		feed.Title = "Al Jazeera"
	}
	return feed, nil
}
