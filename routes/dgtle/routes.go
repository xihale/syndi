package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes exports the dgtle RouteSpecs for registration.
var Routes = []routeutils.RouteSpec{
	dgtleNewsRoute,
	dgtleNewsCategoryRoute,
	dgtleFeedRoute,
}
