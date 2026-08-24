package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all GitLab route specs in this package.
var Routes = []routeutils.RouteSpec{
	gitLabExploreRoute,
	gitLabReleasesRoute,
}
