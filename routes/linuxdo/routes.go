package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all LINUX DO route specs in this package.
var Routes = []routeutils.RouteSpec{
	linuxdoLatestRoute,
	linuxdoTopRoute,
}
