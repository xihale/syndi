package routes

import "github.com/xihale/rsshub-go/internal/routeutils"

// Routes lists all lwn route specs in this package.
var Routes = []routeutils.RouteSpec{
	lwnHeadlinesRoute,
}
