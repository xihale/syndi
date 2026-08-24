package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all nasa-apod route specs in this package.
var Routes = []routeutils.RouteSpec{
	nasaAPODRoute,
}
