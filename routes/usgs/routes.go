package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all usgs route specs in this package.
var Routes = []routeutils.RouteSpec{
	usgsSignificantRoute,
	usgsAllDayRoute,
}
