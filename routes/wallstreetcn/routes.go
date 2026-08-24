package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all wallstreetcn route specs in this package.
var Routes = []routeutils.RouteSpec{
	wscLiveRoute,
	wscLiveCategoryRoute,
	wscNewsRoute,
	wscNewsCategoryRoute,
	wscHotRoute,
	wscHotPeriodRoute,
}
