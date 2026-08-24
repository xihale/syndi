package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes exports the huxiu RouteSpecs for registration.
var Routes = []routeutils.RouteSpec{
	huxiuArticleRoute,
	huxiuChannelRoute,
	huxiuMomentRoute,
}
