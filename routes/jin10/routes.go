package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all jin10 route specs in this package.
var Routes = []routeutils.RouteSpec{
	jin10FlashRoute,
	jin10CategoryRoute,
}
