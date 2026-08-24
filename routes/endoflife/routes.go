package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all endoflife route specs in this package.
var Routes = []routeutils.RouteSpec{
	endOfLifeProductRoute,
}
