package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all Lemmy route specs in this package.
var Routes = []routeutils.RouteSpec{
	lemmyPostsRoute,
}
