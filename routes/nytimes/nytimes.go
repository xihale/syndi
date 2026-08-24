package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const nytimesFeedBase = "https://rss.nytimes.com/services/xml/rss/nyt"

var nytimesRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "NYT Top Stories",
	Example:     "nytimes",
	Maintainers: []string{"xihale"},
	Description: "New York Times top stories from the official RSS feed",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    15 * time.Minute,
	Handler:     NYTimesHandler,
}

var nytimesCategoryRoute = routeutils.RouteSpec{
	Path:        ":category",
	Name:        "NYT Section",
	Example:     "nytimes/technology",
	Maintainers: []string{"xihale"},
	Description: "New York Times section feed from the official RSS service",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("category", "Section name from https://www.nytimes.com/rss, e.g. world, business, technology, sports"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  NYTimesCategoryHandler,
}

// NYTimesHandler handles /nytimes
func NYTimesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), fmt.Sprintf("%s/HomePage.xml", nytimesFeedBase))
	if err != nil {
		return nil, err
	}
	feed.Title = "NYT > Top Stories"
	feed.Link = "https://www.nytimes.com"
	return feed, nil
}

// NYTimesCategoryHandler handles /nytimes/:category
func NYTimesCategoryHandler(c *ctxpkg.Context) (*models.Feed, error) {
	category := strings.TrimSpace(c.Param("category"))
	if category == "" || strings.EqualFold(category, "HomePage") {
		return NYTimesHandler(c)
	}
	feedURL := fmt.Sprintf("%s/%s.xml", nytimesFeedBase, category)
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), feedURL)
	if err != nil {
		return nil, err
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items for NYT section %q; check https://www.nytimes.com/rss for valid sections", category)
	}
	if feed.Title == "" {
		feed.Title = "NYT > " + category
	}
	return feed, nil
}
