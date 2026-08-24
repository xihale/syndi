package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all noaa-alerts route specs in this package.
var Routes = []routeutils.RouteSpec{
	noaaAlertsRoute,
}
