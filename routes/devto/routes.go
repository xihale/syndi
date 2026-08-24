package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all DEV Community route specs in this package.
var Routes = []routeutils.RouteSpec{
	devtoArticlesRoute,
	devtoTagRoute,
}
