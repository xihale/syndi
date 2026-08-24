package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all Dribbble route specs in this package.
var Routes = []routeutils.RouteSpec{
	dribbbleUserRoute,
}
