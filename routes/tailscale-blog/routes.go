package routes

import "github.com/xihale/rsshub-go/internal/routeutils"

// Routes lists all tailscale-blog route specs in this package.
var Routes = []routeutils.RouteSpec{
	tailscaleBlogRootRoute,
}
