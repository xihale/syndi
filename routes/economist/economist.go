package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const economistBaseURL = "https://www.economist.com"

var economistRoute = routeutils.RouteSpec{
	Path:        ":endpoint",
	Name:        "Economist Category",
	Example:     "economist/latest",
	Maintainers: []string{"xihale"},
	Description: "The Economist section feed from the official RSS service. Sections include latest, china, international, business, finance-and-economics, science-and-technology, culture, united-states, europe, asia, graphic-detail, obituary, podcasts",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("endpoint", "Section name, e.g. latest (default), china, business, science-and-technology"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  EconomistHandler,
}

// EconomistHandler handles /economist/:endpoint
func EconomistHandler(c *ctxpkg.Context) (*models.Feed, error) {
	endpoint := strings.Trim(strings.TrimSpace(c.Param("endpoint")), "/")
	if endpoint == "" || endpoint == "-" {
		endpoint = "latest"
	}
	feedURL := fmt.Sprintf("%s/%s/rss.xml", economistBaseURL, endpoint)

	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), feedURL)
	if err != nil {
		return nil, err
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items for Economist section %q; check https://www.economist.com/rss for valid sections", endpoint)
	}
	if feed.Link == "" {
		feed.Link = fmt.Sprintf("%s/%s", economistBaseURL, endpoint)
	}
	return feed, nil
}
