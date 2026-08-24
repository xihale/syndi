package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all Docker Hub route specs in this package.
var Routes = []routeutils.RouteSpec{
	dockerHubTagsRoute,
}
