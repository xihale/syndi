package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes exports the leetcode RouteSpecs for registration.
var Routes = []routeutils.RouteSpec{
	leetCodeDailyCNRoute,
	leetCodeDailyENRoute,
}
