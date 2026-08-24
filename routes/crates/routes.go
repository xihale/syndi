package routes

import "github.com/xihale/rsshub-go/internal/routeutils"

// Routes lists all crates.io route specs in this package.
var Routes = []routeutils.RouteSpec{
	cratesCrateRoute,
}
