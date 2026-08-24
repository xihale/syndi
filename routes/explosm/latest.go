// Package routes implements RSSHub-style routes for Explosm (Cyanide and Happiness).
package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var explosmRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Explosm Comics",
	Example:     "explosm",
	Maintainers: []string{"xihale"},
	Description: "Latest Cyanide and Happiness comics from Explosm.net",
	Categories:  []models.Category{{Name: "picture"}},
	Features:    models.Features{},
	CacheTTL:    2 * time.Hour,
	Handler:     ExplosmHandler,
}

// ExplosmHandler handles /explosm via the site's official RSS feed
// (https://explosm.net/rss.xml; the older /rss and Feedburner feeds are dead).
func ExplosmHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return routeutils.GetFeed(c.Parent(), c.Client(), "https://explosm.net/rss.xml")
}
