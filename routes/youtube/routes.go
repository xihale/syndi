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

var youtubeChannelRoute = routeutils.RouteSpec{
	Path:        "channel/:id",
	Name:        "Channel",
	Example:     "youtube/channel/UCX6OQ3DkcsbYNE6H8uQQuVA",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube channel via the native channel feed",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "YouTube channel id, starts with UC"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubeChannelHandler,
}

// YouTubeChannelHandler handles /youtube/channel/:id.
func YouTubeChannelHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := strings.TrimSpace(c.Param("id"))
	if !strings.HasPrefix(id, "UC") {
		return nil, fmt.Errorf("invalid YouTube channel id %q: must start with UC", id)
	}
	return fetchYouTubeVideos(c, "channel_id="+url.QueryEscape(id))
}

var youtubeUserRoute = routeutils.RouteSpec{
	Path:        "user/:username",
	Name:        "Channel With User Handle",
	Example:     "youtube/user/@JFlaMusic",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube channel resolved from its @handle",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("username", "YouTube @handle (with or without the leading @)"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubeUserHandler,
}

// YouTubeUserHandler handles /youtube/user/:username.
func YouTubeUserHandler(c *ctxpkg.Context) (*models.Feed, error) {
	channelID, err := resolveYouTubeHandle(c, c.Param("username"))
	if err != nil {
		return nil, err
	}
	return fetchYouTubeVideos(c, "channel_id="+url.QueryEscape(channelID))
}

var youtubePlaylistRoute = routeutils.RouteSpec{
	Path:        "playlist/:id",
	Name:        "Playlist",
	Example:     "youtube/playlist/PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z",
	Maintainers: []string{"xihale"},
	Description: "Latest videos of a YouTube playlist via the native playlist feed",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "YouTube playlist id, starts with PL"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YouTubePlaylistHandler,
}

// YouTubePlaylistHandler handles /youtube/playlist/:id.
func YouTubePlaylistHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := strings.TrimSpace(c.Param("id"))
	if !strings.HasPrefix(id, "PL") && !strings.HasPrefix(id, "UU") && !strings.HasPrefix(id, "OL") {
		return nil, fmt.Errorf("invalid YouTube playlist id %q", id)
	}
	return fetchYouTubeVideos(c, "playlist_id="+url.QueryEscape(id))
}

// Routes lists all YouTube route specs in this package.
var Routes = []routeutils.RouteSpec{
	youtubeChannelRoute,
	youtubeUserRoute,
	youtubePlaylistRoute,
}
