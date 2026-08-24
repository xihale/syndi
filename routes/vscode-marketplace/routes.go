package routes

import "github.com/xihale/rsshub-go/internal/routeutils"

// Routes lists all VS Code Marketplace route specs in this package.
var Routes = []routeutils.RouteSpec{
	vscodeExtensionRoute,
}
