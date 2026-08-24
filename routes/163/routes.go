package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all 163 (netease news) route specs in this package.
var Routes = []routeutils.RouteSpec{
	neteaseRankRoute,
	neteaseRankParamsRoute,
}
