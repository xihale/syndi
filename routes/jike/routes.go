package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes exports the jike RouteSpecs for registration.
var Routes = []routeutils.RouteSpec{
	jikeTopicRoute,
	jikeTopicTextRoute,
	jikeUserRoute,
}
