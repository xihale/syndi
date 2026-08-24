package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all V2EX route specs in this package.
var Routes = []routeutils.RouteSpec{
	v2exHotRoute,
	v2exLatestRoute,
	v2exNodeRoute,
	v2exTopicRoute,
}
