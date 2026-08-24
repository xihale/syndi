// Package routes implements RSSHub-style routes for the Steam store.
package routes

import (
	"github.com/xihale/rsshub-go/internal/routeutils"
)

var Routes = []routeutils.RouteSpec{
	steamNewsRoute,
	steamSpecialsRoute,
}
