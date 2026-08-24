package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all MIUI route specs in this package.
// The legacy firmware route is not ported: update.miui.com
// miota-fullrom.php returns an empty body as of 2026-08.
var Routes = []routeutils.RouteSpec{
	miuiCommunityUserRoute,
}
