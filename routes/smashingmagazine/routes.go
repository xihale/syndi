package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all smashingmagazine route specs in this package.
var Routes = []routeutils.RouteSpec{
	latestRoute,
	categoryRoute,
}
