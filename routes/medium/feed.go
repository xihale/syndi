package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var mediumFeedRoute = routeutils.RouteSpec{
	Path:        "feed/:user",
	Name:        "Medium Feed",
	Example:     "medium/feed/zhgchgli",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from a Medium user (native RSS, normalized)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("user", "Medium username, without the leading @"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  MediumFeedHandler,
}

// MediumFeedHandler handles /medium/feed/:user
func MediumFeedHandler(c *ctxpkg.Context) (*models.Feed, error) {
	user := strings.TrimPrefix(strings.TrimSpace(c.Param("user")), "@")
	if user == "" {
		return nil, fmt.Errorf("medium: user is required")
	}

	feedURL := fmt.Sprintf("https://medium.com/feed/@%s", user)
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), feedURL)
	if err != nil {
		return nil, err
	}
	if feed.Title == "" {
		feed.Title = user + "'s Medium"
	}
	if feed.Link == "" {
		feed.Link = fmt.Sprintf("https://medium.com/@%s", user)
	}
	if feed.Description == "" {
		feed.Description = fmt.Sprintf("Latest posts from %s on Medium", user)
	}
	return feed, nil
}
