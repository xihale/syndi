package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var scienceDailyTopRoute = routeutils.RouteSpec{
	Path:        "top",
	Name:        "ScienceDaily Top News",
	Example:     "sciencedaily/top",
	Maintainers: []string{"xihale"},
	Description: "Top science stories featured on ScienceDaily's home page (falls back to the all-news feed)",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     ScienceDailyTopHandler,
}

// ScienceDailyTopHandler handles /sciencedaily/top
func ScienceDailyTopHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feedURL := "https://www.sciencedaily.com/rss/top/science.xml"
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), feedURL)
	if err != nil || len(feed.Items) == 0 {
		// Fallback to the all-news feed when the top feed is unavailable or empty.
		feedURL = "https://www.sciencedaily.com/rss/all.xml"
		feed, err = routeutils.GetFeed(c.Parent(), c.Client(), feedURL)
		if err != nil {
			return nil, err
		}
	}
	feed.Title = "ScienceDaily Top News"
	feed.Link = "https://www.sciencedaily.com/"
	feed.Description = "Top science news from ScienceDaily"
	return feed, nil
}
