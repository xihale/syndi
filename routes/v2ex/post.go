package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var v2exPostRoute = routeutils.RouteSpec{
	Path:        "post/:postid",
	Name:        "V2EX Post Replies",
	Example:     "v2ex/post/584403",
	Maintainers: []string{"xihale"},
	Description: "Replies of a V2EX post (upstream-compatible alias of /v2ex/topic/:id)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("postid", "Numeric post ID, e.g. 584403"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  V2EXPostHandler,
}

// V2EXPostHandler handles /v2ex/post/:postid. It serves the same replies
// feed as /v2ex/topic/:id through the shared implementation.
func V2EXPostHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return v2exTopicRepliesFeed(c, c.Param("postid"))
}
