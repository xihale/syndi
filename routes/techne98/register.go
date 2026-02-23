package routes

import (
	"sync"

	"github.com/xihale/rsshub-go/internal/routeutils"
)

var registerOnce sync.Once

// RegisterRoutes registers all techne98 routes explicitly.
func RegisterRoutes() {
	registerOnce.Do(func() {
		routeutils.MustRegisterRoute(techne98BlogRoute)
	})
}
