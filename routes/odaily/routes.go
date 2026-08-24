package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	"github.com/xihale/syndi/pkg/models"
)

// Routes lists all odaily routes registered under /odaily.
//
// Not ported from upstream (with reasons):
//   - /:id? category feed   the legacy feed_id taxonomy (280 最新, 331 DeFi…)
//     died with the old api/pp backend; the new post/page API has no
//     equivalent filter yet
//   - /search/news/:keyword  the new web frontend ships no public search API
//     (no search endpoint found in the deployed JS bundles)
//   - /user/:id              the new /service/feed_stream/user endpoint is
//     gone; user pages are client-rendered without a discoverable feed API
var Routes = []routeutils.RouteSpec{
	{
		Path:        "",
		Name:        "Latest Posts",
		URL:         "https://www.odaily.news",
		Example:     "odaily",
		Maintainers: []string{"xihale"},
		Description: "Latest Odaily (星球日报) featured articles with full content",
		Categories:  []models.Category{{Name: "new-media"}},
		Features:    models.Features{SupportRadar: true},
		Parameters: []models.Parameter{
			routeutils.OptionalParam("limit", "default 15, max 25; each item costs one detail request"),
		},
		CacheTTL: 15 * time.Minute,
		Handler:  OdailyPostHandler,
	},
	{
		Path:        "newsflash",
		Name:        "Newsflash",
		URL:         "https://www.odaily.news/newsflash",
		Example:     "odaily/newsflash",
		Maintainers: []string{"xihale"},
		Description: "Odaily rolling crypto newsflashes (快讯)",
		Categories:  []models.Category{{Name: "new-media"}},
		Features:    models.Features{SupportRadar: true},
		Parameters: []models.Parameter{
			routeutils.OptionalParam("limit", "default 30, max 100"),
		},
		CacheTTL: 10 * time.Minute,
		Handler:  OdailyNewsflashHandler,
	},
	{
		Path:        "hot",
		Name:        "Hot Posts Weekly",
		URL:         "https://www.odaily.news/hot",
		Example:     "odaily/hot",
		Maintainers: []string{"xihale"},
		Description: "Odaily weekly hot post ranking with full content",
		Categories:  []models.Category{{Name: "new-media"}},
		Features:    models.Features{SupportRadar: true},
		Parameters: []models.Parameter{
			routeutils.OptionalParam("limit", "default 10, max 25"),
		},
		CacheTTL: time.Hour,
		Handler:  OdailyHotHandler,
	},
	{
		Path:        "hot/:period",
		Name:        "Hot Posts",
		URL:         "https://www.odaily.news/hot",
		Example:     "odaily/hot/daily",
		Maintainers: []string{"xihale"},
		Description: "Odaily daily or weekly hot post ranking with full content",
		Categories:  []models.Category{{Name: "new-media"}},
		Features:    models.Features{SupportRadar: true},
		Parameters: []models.Parameter{
			routeutils.RequiredParam("period", "Ranking window: daily or weekly"),
			routeutils.OptionalParam("limit", "default 10, max 25"),
		},
		CacheTTL: time.Hour,
		Handler:  OdailyHotHandler,
	},
}
