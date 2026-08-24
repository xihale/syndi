package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all weibo route specs in this package.
var Routes = []routeutils.RouteSpec{
	weiboUserRoute,
	weiboHotSearchRoute,
}
