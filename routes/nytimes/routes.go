package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all NYT route specs in this package.
var Routes = []routeutils.RouteSpec{
	nytimesRoute,
	nytimesCategoryRoute,
}
