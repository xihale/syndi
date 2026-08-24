package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all coolapk route specs in this package.
var Routes = []routeutils.RouteSpec{
	coolapkHotRoute,
	coolapkHotTypeRoute,
	coolapkToutiaoRoute,
	coolapkToutiaoTypeRoute,
}
