package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all pubmed route specs in this package.
var Routes = []routeutils.RouteSpec{
	pubmedSearchRoute,
}
