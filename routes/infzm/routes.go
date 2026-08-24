package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all infzm route specs in this package.
var Routes = []routeutils.RouteSpec{
	infzmHotRoute,
	infzmChannelRoute,
}
