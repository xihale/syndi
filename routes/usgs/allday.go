package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var usgsAllDayRoute = routeutils.RouteSpec{
	Path:        "all-day",
	Name:        "USGS All Earthquakes Past Day",
	Example:     "usgs/all-day",
	Maintainers: []string{"xihale"},
	Description: "All detected earthquakes worldwide in the past day, from the USGS Earthquake Hazards Program",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     USGSAllDayHandler,
}

// USGSAllDayHandler handles /usgs/all-day
func USGSAllDayHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return buildUSGSFeed(
		c,
		usgsBaseURL+"all_day.geojson",
		"USGS All Earthquakes, Past Day",
		"All detected earthquakes worldwide in the past day, from the USGS Earthquake Hazards Program",
	)
}
