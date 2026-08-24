package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all Mastodon route specs in this package.
var Routes = []routeutils.RouteSpec{
	mastodonAccountRoute,
	mastodonTimelineRoute,
}
