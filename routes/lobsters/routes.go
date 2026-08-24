package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all Lobsters route specs in this package.
var Routes = []routeutils.RouteSpec{
	lobstersHotRoute,
	lobstersNewestRoute,
	lobstersTagRoute,
}
