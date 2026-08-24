package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all douban route specs in this package.
var Routes = []routeutils.RouteSpec{
	doubanMoviePlayingRoute,
	doubanMoviePlayingScoreRoute,
	doubanMovieWeeklyRoute,
}
