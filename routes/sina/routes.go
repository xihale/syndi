package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all sina route specs in this package.
var Routes = []routeutils.RouteSpec{
	sinaRollNewsRoute,
	sinaRollNewsSectionRoute,
}
