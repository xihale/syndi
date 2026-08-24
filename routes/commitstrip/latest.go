// Package routes implements RSSHub-style routes for CommitStrip.
package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var commitStripRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "CommitStrip",
	Example:     "commitstrip",
	Maintainers: []string{"xihale"},
	Description: "Latest CommitStrip comics about developer life",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{},
	CacheTTL:    2 * time.Hour,
	Handler:     CommitStripHandler,
}

// CommitStripHandler handles /commitstrip via the official WordPress feed.
func CommitStripHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return routeutils.GetFeed(c.Parent(), c.Client(), "https://www.commitstrip.com/en/feed/")
}
