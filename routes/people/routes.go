package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all people route specs in this package.
var Routes = []routeutils.RouteSpec{
	peopleHeadlinesRoute,
	peopleChannelRoute,
}
