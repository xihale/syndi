package routes

import "github.com/xihale/rsshub-go/internal/routeutils"

// Routes lists all npm route specs in this package.
var Routes = []routeutils.RouteSpec{
	npmPackageRoute,
}
