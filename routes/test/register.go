package test

import (
	"sync"

	"github.com/xihale/rsshub-go/internal/routeutils"
)

var registerOnce sync.Once

// RegisterRoutes registers all test routes explicitly.
func RegisterRoutes() {
	registerOnce.Do(func() {
		routeutils.MustRegisterRoute(cacheTestRoute)
	})
}
