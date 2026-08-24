package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var fedoraMagazineRootRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Fedora Magazine",
	Example:     "fedora-magazine",
	Maintainers: []string{"xihale"},
	Description: "Latest articles from Fedora Magazine (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     FedoraMagazineRootHandler,
}

// FedoraMagazineRootHandler handles /fedora-magazine
func FedoraMagazineRootHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://fedoramagazine.org/feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "Fedora Magazine"
	feed.Link = "https://fedoramagazine.org/"
	feed.Description = "The Fedora Community blog"
	return feed, nil
}
