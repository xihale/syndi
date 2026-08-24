package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var omgubuntuRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "OMG! Ubuntu! Feed",
	Example:     "omgubuntu",
	Maintainers: []string{"xihale"},
	Description: "Latest Ubuntu news from OMG! Ubuntu! (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     OMGUbuntuHandler,
}

// OMGUbuntuHandler handles /omgubuntu
func OMGUbuntuHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://www.omgubuntu.co.uk/feed")
	if err != nil {
		return nil, err
	}
	feed.Title = "OMG! Ubuntu!"
	feed.Link = "https://www.omgubuntu.co.uk/"
	feed.Description = "OMG! Ubuntu! - Ubuntu and GNOME news, apps and tutorials"
	return feed, nil
}
