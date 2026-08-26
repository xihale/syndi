package routes

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// Gin has no optional path segments, so every upstream ":param?" tail is
// registered twice: once bare and once as a catch-all twin sharing the same
// handler and cache TTL.

const (
	youtubeParamDesc = `key=value pairs joined by '&': embed (1/0, default 1) render an embedded player instead of a thumbnail image; a bare "/embed" also disables it; filterShorts (1/0, default 0) restrict channel uploads to the shorts-free playlist`
	youtubeIDParam   = "YouTube channel id, starts with UC"
)

var youtubeChannelRoute = routeutils.RouteSpec{
	Path:        "channel/:id",
	Name:        "Channel",
	Example:     "youtube/channel/UCX6OQ3DkcsbYNE6H8uQQuVA",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube channel via the official channel feed",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", youtubeIDParam),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubeChannelHandler,
}

var youtubeChannelParamsRoute = routeutils.RouteSpec{
	Path:        "channel/:id/*routeParams",
	Name:        "Channel",
	Example:     "youtube/channel/UCX6OQ3DkcsbYNE6H8uQQuVA/embed=0&filterShorts=0",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube channel with extra switches",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", youtubeIDParam),
		routeutils.OptionalParam("routeParams", youtubeParamDesc),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubeChannelHandler,
}

var youtubeUserRoute = routeutils.RouteSpec{
	Path:        "user/:username",
	Name:        "Channel With User Handle",
	Example:     "youtube/user/@JFlaMusic",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube channel resolved from its @handle or legacy username",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("username", "YouTube @handle or legacy username"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubeUserHandler,
}

var youtubeUserParamsRoute = routeutils.RouteSpec{
	Path:        "user/:username/*routeParams",
	Name:        "Channel With User Handle",
	Example:     "youtube/user/@JFlaMusic/embed=0",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube channel from its @handle with extra switches",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("username", "YouTube @handle or legacy username"),
		routeutils.OptionalParam("routeParams", youtubeParamDesc),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubeUserHandler,
}

var youtubeCustomRoute = routeutils.RouteSpec{
	Path:        "c/:username",
	Name:        "Channel With Custom URL",
	Example:     "youtube/c/TED",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube channel resolved from its legacy /c/ vanity URL",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("username", "YouTube custom URL name (youtube.com/c/<name>)"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubeCustomHandler,
}

var youtubeCustomParamsRoute = routeutils.RouteSpec{
	Path:        "c/:username/*routeParams",
	Name:        "Channel With Custom URL",
	Example:     "youtube/c/TED/embed=0",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube channel from its /c/ vanity URL with extra switches",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("username", "YouTube custom URL name (youtube.com/c/<name>)"),
		routeutils.OptionalParam("routeParams", youtubeParamDesc),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubeCustomHandler,
}

var youtubePlaylistRoute = routeutils.RouteSpec{
	Path:        "playlist/:id",
	Name:        "Playlist",
	Example:     "youtube/playlist/PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube playlist via the official playlist feed",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "YouTube playlist id, starts with PL (uploads feeds start with UU)"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubePlaylistHandler,
}

var youtubePlaylistParamsRoute = routeutils.RouteSpec{
	Path:        "playlist/:id/*routeParams",
	Name:        "Playlist",
	Example:     "youtube/playlist/PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z/embed=0",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube playlist with extra switches",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "YouTube playlist id, starts with PL (uploads feeds start with UU)"),
		routeutils.OptionalParam("routeParams", `key=value pairs joined by '&': embed (1/0, default 1) render an embedded player instead of a thumbnail image`),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubePlaylistHandler,
}

// Routes lists all YouTube route specs in this package.
var Routes = []routeutils.RouteSpec{
	youtubeChannelRoute,
	youtubeChannelParamsRoute,
	youtubeUserRoute,
	youtubeUserParamsRoute,
	youtubeCustomRoute,
	youtubeCustomParamsRoute,
	youtubePlaylistRoute,
	youtubePlaylistParamsRoute,
}

// YouTubeChannelHandler handles /youtube/channel/:id[/*routeParams].
func YouTubeChannelHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := strings.TrimSpace(c.Param("id"))
	if !strings.HasPrefix(id, "UC") {
		return nil, fmt.Errorf("invalid YouTube channel id %q: must start with UC", id)
	}
	return fetchYouTubeChannelFeed(c, id, parseYouTubeRouteParams(c.Param("routeParams")))
}

// YouTubeUserHandler handles /youtube/user/:username[/*routeParams]. Legacy
// usernames are served by the official ?user= feed; anything else is resolved
// through its @handle page first.
func YouTubeUserHandler(c *ctxpkg.Context) (*models.Feed, error) {
	username := strings.TrimPrefix(strings.TrimSpace(c.Param("username")), "@")
	if username == "" {
		return nil, fmt.Errorf("empty YouTube username")
	}
	params := parseYouTubeRouteParams(c.Param("routeParams"))

	if strings.HasPrefix(username, "UC") {
		return fetchYouTubeChannelFeed(c, username, params)
	}
	if !params.filterShorts {
		if feed, err := fetchYouTubeVideos(c, "user="+url.QueryEscape(username), params); err == nil {
			return feed, nil
		}
	}
	channelID, err := resolveYouTubeHandle(c, username)
	if err != nil {
		return nil, err
	}
	return fetchYouTubeChannelFeed(c, channelID, params)
}

// YouTubeCustomHandler handles /youtube/c/:username[/*routeParams].
func YouTubeCustomHandler(c *ctxpkg.Context) (*models.Feed, error) {
	name := strings.TrimSpace(c.Param("username"))
	if name == "" {
		return nil, fmt.Errorf("empty YouTube custom URL name")
	}
	channelID, err := resolveYouTubeCustomURL(c, name)
	if err != nil {
		return nil, err
	}
	return fetchYouTubeChannelFeed(c, channelID, parseYouTubeRouteParams(c.Param("routeParams")))
}

// YouTubePlaylistHandler handles /youtube/playlist/:id[/*routeParams].
func YouTubePlaylistHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := strings.TrimSpace(c.Param("id"))
	for _, prefix := range []string{"PL", "UU", "OL"} {
		if strings.HasPrefix(id, prefix) {
			return fetchYouTubeVideos(c, "playlist_id="+url.QueryEscape(id), parseYouTubeRouteParams(c.Param("routeParams")))
		}
	}
	return nil, fmt.Errorf("invalid YouTube playlist id %q", id)
}
