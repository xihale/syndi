package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all juejin route specs in this package.
var Routes = []routeutils.RouteSpec{
	juejinCategoryRoute,
	juejinTrendingRoute,
	juejinColumnRoute,
	juejinTagRoute,
	juejinPostsRoute,
	juejinBooksRoute,
	juejinCollectionRoute,
	juejinDynamicRoute,
	juejinPinsRoute,
	juejinPinsTypeRoute,
}
