package routes

import "github.com/xihale/rsshub-go/internal/routeutils"

// Routes lists all debian-news route specs in this package.
var Routes = []routeutils.RouteSpec{
	debianNewsRootRoute,
}
