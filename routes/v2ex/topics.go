package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var v2exTopicsRoute = routeutils.RouteSpec{
	Path:        "topics/:type",
	Name:        "V2EX Topics by Type",
	Example:     "v2ex/topics/latest",
	Maintainers: []string{"xihale"},
	Description: "V2EX topic list by type (hot or latest, mirroring the upstream conversion)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("type", "Topic list type: hot or latest"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  V2EXTopicsHandler,
}

// V2EXTopicsHandler handles /v2ex/topics/:type.
// The legacy show.json?tab_name= parameter no longer works upstream, so
// arbitrary tabs should use /v2ex/tab/:tabid and nodes /v2ex/node/:name.
func V2EXTopicsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	switch strings.ToLower(strings.TrimSpace(c.Param("type"))) {
	case "hot":
		return fetchV2EXTopics(c, v2exAPI("/api/topics/hot.json"), "V2EX Hot Topics")
	case "latest":
		return fetchV2EXTopics(c, v2exAPI("/api/topics/latest.json"), "V2EX Latest Topics")
	default:
		return nil, fmt.Errorf("unsupported topics type %q; expected hot or latest", c.Param("type"))
	}
}
