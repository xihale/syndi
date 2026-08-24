package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all Hacker News route specs in this package.
var Routes = []routeutils.RouteSpec{
	hackerNewsStoriesRoute,
}
