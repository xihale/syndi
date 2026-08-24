package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all toutiao route specs in this package.
var Routes = []routeutils.RouteSpec{
	toutiaoUserRoute,
}
