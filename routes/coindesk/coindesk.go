package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var coinDeskRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "CoinDesk",
	Example:     "coindesk",
	Maintainers: []string{"xihale"},
	Description: "Latest crypto and finance news from CoinDesk (native RSS, normalized)",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     CoinDeskHandler,
}

// CoinDeskHandler handles /coindesk
func CoinDeskHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://www.coindesk.com/arc/outboundfeeds/rss/")
	if err != nil {
		return nil, err
	}
	feed.Title = "CoinDesk"
	feed.Link = "https://www.coindesk.com/"
	feed.Description = "Latest crypto and finance news from CoinDesk"
	return feed, nil
}
