package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all Twitch route specs in this package.
var Routes = []routeutils.RouteSpec{
	twitchLiveRoute,
	twitchScheduleRoute,
	twitchVideoRoute,
	twitchVideoFilteredRoute,
}
