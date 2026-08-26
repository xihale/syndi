package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all sspai route specs in this package.
var Routes = []routeutils.RouteSpec{
	sspaiRoute,
	sspaiIndexRoute,
	sspaiMatrixRoute,
	sspaiColumnRoute,
	sspaiAuthorRoute,
	sspaiTagRoute,
	sspaiTopicsRoute,
	sspaiBookmarksRoute,
}
