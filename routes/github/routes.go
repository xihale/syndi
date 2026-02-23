package routes

import "github.com/xihale/rsshub-go/internal/routeutils"

// Routes lists all GitHub route specs in this package.
var Routes = []routeutils.RouteSpec{
	gitHubReposRoute,
	gitHubTrendingRoute,
	gitHubUserReposRoute,
}
