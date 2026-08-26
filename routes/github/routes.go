package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all GitHub route specs in this package.
var Routes = []routeutils.RouteSpec{
	gitHubReposRoute,
	gitHubStarsRoute,
	gitHubActivityRoute,
	gitHubTrendingRoute,
	gitHubUserReposRoute,
	gitHubCommitsRoute,
	gitHubFileRoute,
	gitHubIssuesRoute,
	gitHubIssueRoute,
	gitHubIssueStateRoute,
	gitHubIssueLabelsRoute,
	gitHubPullRoute,
	gitHubPullStateRoute,
	gitHubPullLabelsRoute,
	gitHubCommentsRoute,
	gitHubCommentsNumberRoute,
	gitHubBranchesRoute,
	gitHubTopicsRoute,
	gitHubTopicsQSRoute,
	gitHubGistsRoute,
}
