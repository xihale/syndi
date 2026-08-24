package routes

import "github.com/xihale/rsshub-go/internal/routeutils"

// Routes lists all stackexchange route specs in this package.
var Routes = []routeutils.RouteSpec{
	stackExchangeQuestionsRoute,
}
