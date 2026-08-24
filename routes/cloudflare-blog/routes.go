package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all cloudflare-blog route specs in this package.
var Routes = []routeutils.RouteSpec{
	cloudflareBlogRoute,
}
