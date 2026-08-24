package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all baidu route specs in this package.
var Routes = []routeutils.RouteSpec{
	baiduTopRoute,
	baiduTopBoardRoute,
}
