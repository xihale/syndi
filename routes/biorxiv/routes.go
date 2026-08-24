package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all biorxiv/medrxiv route specs in this package.
var Routes = []routeutils.RouteSpec{
	biorxivLatestRoute,
	medrxivLatestRoute,
}
