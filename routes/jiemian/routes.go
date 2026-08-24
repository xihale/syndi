package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes exports the jiemian RouteSpecs for registration.
var Routes = []routeutils.RouteSpec{
	jiemianHomeRoute,
	jiemianListsRoute,
}
