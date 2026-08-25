package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all GitHub route specs in this package.
var Routes = []routeutils.RouteSpec{
	gitHubReposRoute,
	gitHubActivityRoute,
	gitHubTrendingRoute,
	gitHubUserReposRoute,
	gitHubCommitsRoute,
	gitHubIssuesRoute,
	gitHubPullRoute,
	gitHubGistsRoute,
}
