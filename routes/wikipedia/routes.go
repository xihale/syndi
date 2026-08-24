package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all wikipedia route specs in this package.
var Routes = []routeutils.RouteSpec{
	wikipediaOnThisDayRoute,
	wikipediaFeaturedRoute,
}
