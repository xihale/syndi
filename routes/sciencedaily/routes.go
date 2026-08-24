package routes

import "github.com/xihale/rsshub-go/internal/routeutils"

// Routes lists all sciencedaily route specs in this package.
var Routes = []routeutils.RouteSpec{
	scienceDailyTopRoute,
}
