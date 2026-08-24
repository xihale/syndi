package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var usgsSignificantRoute = routeutils.RouteSpec{
	Path:        "significant",
	Name:        "USGS Significant Earthquakes",
	Example:     "usgs/significant",
	Maintainers: []string{"xihale"},
	Description: "Significant earthquakes worldwide in the past day, from the USGS Earthquake Hazards Program",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     USGSSignificantHandler,
}

// USGSSignificantHandler handles /usgs/significant
func USGSSignificantHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return buildUSGSFeed(
		c,
		usgsBaseURL+"significant_day.geojson",
		"USGS Significant Earthquakes, Past Day",
		"Significant earthquakes worldwide in the past day, from the USGS Earthquake Hazards Program",
	)
}
