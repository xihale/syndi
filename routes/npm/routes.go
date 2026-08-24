package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all npm route specs in this package.
var Routes = []routeutils.RouteSpec{
	npmPackageRoute,
	npmSearchRoute,
}
