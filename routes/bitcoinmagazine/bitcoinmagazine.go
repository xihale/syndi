package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var bitcoinMagazineRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Bitcoin Magazine",
	Example:     "bitcoinmagazine",
	Maintainers: []string{"xihale"},
	Description: "Latest Bitcoin news and analysis from Bitcoin Magazine (native RSS, normalized)",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     BitcoinMagazineHandler,
}

// BitcoinMagazineHandler handles /bitcoinmagazine
func BitcoinMagazineHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://bitcoinmagazine.com/feed")
	if err != nil {
		return nil, err
	}
	feed.Title = "Bitcoin Magazine"
	feed.Link = "https://bitcoinmagazine.com/"
	feed.Description = "Latest Bitcoin news and analysis from Bitcoin Magazine"
	return feed, nil
}
