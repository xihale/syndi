package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var phoronixNewsRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Phoronix News",
	Example:     "phoronix",
	Maintainers: []string{"xihale"},
	Description: "Latest Linux hardware and kernel news from Phoronix (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     PhoronixNewsHandler,
}

// PhoronixNewsHandler handles /phoronix
func PhoronixNewsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://www.phoronix.com/rss.php")
	if err != nil {
		return nil, err
	}
	feed.Title = "Phoronix"
	feed.Link = "https://www.phoronix.com/"
	feed.Description = "Phoronix Linux hardware reviews, kernel benchmarks and open-source news"
	return feed, nil
}
