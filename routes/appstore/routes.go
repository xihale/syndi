package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all App Store route specs in this package.
var Routes = []routeutils.RouteSpec{
	appstoreXianmianRoute,
	appstorePriceRoute,
}
