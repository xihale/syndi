package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all caixin route specs in this package.
var Routes = []routeutils.RouteSpec{
	caixinLatestRoute,
	caixinCategoryRoute,
}
