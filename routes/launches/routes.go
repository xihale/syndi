package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all launches route specs in this package.
var Routes = []routeutils.RouteSpec{
	launchesUpcomingRoute,
}
