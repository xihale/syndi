package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all nodejs route specs in this package.
var Routes = []routeutils.RouteSpec{
	nodejsBlogRoute,
	nodejsReleaseRoute,
}
