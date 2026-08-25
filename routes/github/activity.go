package routes

import (
	"fmt"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitHubActivityRoute = routeutils.RouteSpec{
	Path:        "activity/:user",
	Name:        "GitHub User Activities",
	Example:     "github/activity/DIYgod",
	Maintainers: []string{"xihale"},
	Description: "User activities on GitHub, based on the GitHub official Atom feed",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("user", "GitHub username"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  GitHubActivityHandler,
}

// GitHubActivityHandler handles /github/activity/:user
func GitHubActivityHandler(c *ctxpkg.Context) (*models.Feed, error) {
	user := c.Param("user")

	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), fmt.Sprintf("https://github.com/%s.atom", user))
	if err != nil {
		return nil, err
	}
	feed.Title = fmt.Sprintf("%s's GitHub activities", user)
	feed.Description = fmt.Sprintf("Public activities of %s on GitHub", user)
	return feed, nil
}
