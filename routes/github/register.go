package routes

import (
	"sync"

	"github.com/xihale/rsshub-go/internal/routeutils"
)

var registerOnce sync.Once

// RegisterRoutes registers all GitHub routes explicitly.
func RegisterRoutes() {
	registerOnce.Do(func() {
		routeutils.MustRegisterRoute(gitHubReposRoute)
		routeutils.MustRegisterRoute(gitHubTrendingRoute)
		routeutils.MustRegisterRoute(gitHubUserReposRoute)
	})
}
