package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	"github.com/xihale/syndi/pkg/models"
)

// Routes lists all pixiv routes registered under /pixiv.
//
// Not ported from upstream (with reasons):
//   - ranking/:mode       the legacy ranking.php?format=json API now answers
//     404 ("ランキングが見つかりませんでした") and the React SSR page only
//     embeds a handful of preview entries, not the full ranked list
//   - illustfollow        personal followed-artists feed, AppAPI-only upstream
//   - bookmarks           personal bookmarks feed, AppAPI-only upstream
//   - novel-series / user novels
//
// All routes require PIXIV_COOKIES (see common.go).
var Routes = []routeutils.RouteSpec{
	{
		Path:        "user/:id",
		Name:        "User Activity",
		URL:         "https://www.pixiv.net",
		Example:     "pixiv/user/15288095",
		Maintainers: []string{"xihale"},
		Description: "Latest artworks of a pixiv user by numeric user id",
		Categories:  []models.Category{{Name: "social-media"}, {Name: "picture"}},
		Features:    models.Features{SupportRadar: true, NSFW: true, EnvDeps: []string{pixivCookiesEnv}},
		Parameters: []models.Parameter{
			routeutils.RequiredParam("id", "Numeric user id from the profile URL"),
			routeutils.OptionalParam("limit", "default 10, max 30; each item costs one detail request"),
		},
		CacheTTL: 30 * time.Minute,
		Handler:  PixivUserHandler,
	},
	{
		Path:        "search/:keyword",
		Name:        "Artwork Search",
		URL:         "https://www.pixiv.net",
		Example:     "pixiv/search/GenshinImpact",
		Maintainers: []string{"xihale"},
		Description: "Newest pixiv artwork search results for a keyword or tag",
		Categories:  []models.Category{{Name: "social-media"}, {Name: "picture"}},
		Features:    models.Features{NSFW: true, EnvDeps: []string{pixivCookiesEnv}},
		Parameters: []models.Parameter{
			routeutils.RequiredParam("keyword", "Search keyword or tag"),
			routeutils.OptionalParam("limit", "default 20, max 60"),
		},
		CacheTTL: 30 * time.Minute,
		Handler:  PixivSearchHandler,
	},
	{
		Path:        "novel-search/:keyword",
		Name:        "Novel Search",
		URL:         "https://www.pixiv.net",
		Example:     "pixiv/novel-search/原神",
		Maintainers: []string{"xihale"},
		Description: "Newest pixiv novel search results for a keyword or tag",
		Categories:  []models.Category{{Name: "social-media"}},
		Features:    models.Features{NSFW: true, EnvDeps: []string{pixivCookiesEnv}},
		Parameters: []models.Parameter{
			routeutils.RequiredParam("keyword", "Search keyword or tag"),
			routeutils.OptionalParam("limit", "default 20, max 60"),
		},
		CacheTTL: 30 * time.Minute,
		Handler:  PixivNovelSearchHandler,
	},
}
