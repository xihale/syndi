package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

// medrxivLatestRoute lives in this package because both preprint servers share
// the bioRxiv API; the absolute path escapes the /biorxiv base path.
var medrxivLatestRoute = routeutils.RouteSpec{
	Path:        "/medrxiv/latest",
	Name:        "medRxiv Latest Papers",
	Example:     "medrxiv/latest",
	Maintainers: []string{"xihale"},
	Description: "Latest preprints posted to medRxiv in the last 7 days",
	Categories:  []models.Category{{Name: "study"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     MedrxivLatestHandler,
}

// MedrxivLatestHandler handles /medrxiv/latest
func MedrxivLatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return biorxivMedrxivLatest(c, "medrxiv", "medRxiv", "https://www.medrxiv.org/")
}
