package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all douban route specs in this package.
var Routes = []routeutils.RouteSpec{
	doubanMoviePlayingRoute,
	doubanMoviePlayingScoreRoute,

	doubanMovieComingRoute,
	doubanMovieLaterRoute,
	doubanMovieWeeklyRoute,
	doubanMovieWeeklyTypeRoute,
	doubanMovieUSBoxRoute,

	doubanGroupRoute,
	doubanGroupTypeRoute,
	doubanDoulistRoute,
	doubanTopicRoute,
	doubanTopicSortRoute,
	doubanExploreRoute,

	doubanBookLatestRoute,
	doubanBookLatestTypeRoute,
	doubanMusicLatestRoute,
	doubanMusicLatestAreaRoute,

	// Community extras
	doubanEventHotRoute,
	doubanTVComingRoute,
	doubanTVComingSortRoute,
	doubanJobsRoute,
	doubanChannelRoute,
	doubanChannelNavRoute,
}
