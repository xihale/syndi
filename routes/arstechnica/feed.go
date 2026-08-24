package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var arstechnicaRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Ars Technica Feed",
	Example:     "arstechnica",
	Maintainers: []string{"xihale"},
	Description: "Latest news from Ars Technica (native RSS, normalized)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     ArsTechnicaHandler,
}

// ArsTechnicaHandler handles /arstechnica
func ArsTechnicaHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://feeds.arstechnica.com/arstechnica/index")
	if err != nil {
		return nil, err
	}
	feed.Title = "Ars Technica"
	feed.Link = "https://arstechnica.com/"
	feed.Description = "Ars Technica - serving the technologist for over a decade"
	return feed, nil
}
