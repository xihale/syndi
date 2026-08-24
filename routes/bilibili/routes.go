package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all bilibili route specs in this package.
var Routes = []routeutils.RouteSpec{
	bilibiliPopularRoute,
	bilibiliHotSearchRoute,
	bilibiliRankingRoute,
	bilibiliRankingZoneRoute,
}
