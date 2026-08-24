package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all stackexchange route specs in this package.
var Routes = []routeutils.RouteSpec{
	stackExchangeQuestionsRoute,
}
