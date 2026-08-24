package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var nineToFiveGoogleRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "9to5Google",
	Example:     "nine-to-five-google",
	Maintainers: []string{"xihale"},
	Description: "Latest Google news from 9to5Google (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     NineToFiveGoogleHandler,
}

// NineToFiveGoogleHandler handles /nine-to-five-google
func NineToFiveGoogleHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://9to5google.com/feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "9to5Google"
	feed.Link = "https://9to5google.com/"
	feed.Description = "Latest Google news from 9to5Google"
	return feed, nil
}
