package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var dribbbleUserRoute = routeutils.RouteSpec{
	Path:        "user/:name",
	Name:        "User Shots",
	Example:     "dribbble/user/google",
	Maintainers: []string{"xihale"},
	Description: "Latest shots of a Dribbble user or team",
	Categories:  []models.Category{{Name: "picture"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("name", "Dribbble username, from the profile page URL"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  DribbbleUserHandler,
}

// DribbbleUserHandler handles /dribbble/user/:name.
func DribbbleUserHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return dribbbleUserShots(c, c.Param("name"))
}
