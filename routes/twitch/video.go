package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// broadcast type filter -> GraphQL BroadcastType enum (empty = no filter).
var twitchVideoFilters = map[string]string{
	"archive":    "ARCHIVE",
	"highlights": "HIGHLIGHT",
	"all":        "",
}

var twitchVideoRoute = routeutils.RouteSpec{
	Path:        "video/:login",
	Name:        "Channel Videos",
	Example:     "twitch/video/riotgames",
	Maintainers: []string{"xihale"},
	Description: "Latest VODs of a Twitch channel (all types)",
	Categories:  []models.Category{{Name: "live"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("login", "Twitch username"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  TwitchVideoHandler,
}

var twitchVideoFilteredRoute = routeutils.RouteSpec{
	Path:        "video/:login/:filter",
	Name:        "Channel Videos by Type",
	Example:     "twitch/video/riotgames/highlights",
	Maintainers: []string{"xihale"},
	Description: "Latest VODs of a Twitch channel filtered by type (archive or highlights)",
	Categories:  []models.Category{{Name: "live"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("login", "Twitch username"),
		routeutils.RequiredParam("filter", "video type: archive, highlights or all"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  TwitchVideoHandler,
}

// TwitchVideoHandler handles /twitch/video/:login/:filter.
func TwitchVideoHandler(c *ctxpkg.Context) (*models.Feed, error) {
	login := strings.ToLower(c.Param("login"))
	filter := strings.ToLower(strings.TrimSpace(c.Param("filter")))
	if filter == "" {
		filter = "all"
	}
	broadcastType, ok := twitchVideoFilters[filter]
	if !ok {
		return nil, fmt.Errorf("unsupported video filter %q, choose from archive | highlights | all", filter)
	}

	variables := map[string]any{"login": login}
	query := `query UserVideos($login: String!`
	if broadcastType != "" {
		query += `, $type: BroadcastType!`
		variables["type"] = broadcastType
	}
	query += `) {
		user(login: $login) {
			id displayName profileImageURL(width: 70)
			videos(first: 20`
	if broadcastType != "" {
		query += `, type: $type`
	}
	query += `, sort: TIME) {
			edges { node { id title publishedAt lengthSeconds previewThumbnailURL(height: 315, width: 560) game { displayName } } }
		}
	}
}`

	user, err := twitchQuery(c, gqlCall{Query: query, Variables: variables})
	if err != nil {
		return nil, err
	}
	user, err = twitchRequireUser(user, login)
	if err != nil {
		return nil, err
	}

	link := twitchLoginLink(login)
	feed := routeutils.NewFeed(
		fmt.Sprintf("Twitch - %s - Videos (%s)", user.DisplayName, filter),
		link,
		fmt.Sprintf("Latest videos of %s", user.DisplayName),
	)
	for _, edge := range user.Videos.Edges {
		v := edge.Node
		if v.Title == "" || v.ID == "" {
			continue
		}
		desc := ""
		if v.Preview != "" {
			desc = fmt.Sprintf(`<img style="max-width:100%%;" src="%s"><br>`, html.EscapeString(v.Preview))
		}
		parts := make([]string, 0, 2)
		if v.Game != nil && v.Game.DisplayName != "" {
			parts = append(parts, "Category: "+html.EscapeString(v.Game.DisplayName))
		}
		if v.LengthSeconds > 0 {
			parts = append(parts, fmt.Sprintf("Duration: %02d:%02d:%02d", v.LengthSeconds/3600, v.LengthSeconds%3600/60, v.LengthSeconds%60))
		}
		if len(parts) > 0 {
			desc += strings.Join(parts, "<br>")
		}
		videoLink := fmt.Sprintf("https://www.twitch.tv/videos/%s", v.ID)
		item := routeutils.NewItem(v.Title, videoLink, desc, v.PublishedAt)
		item.GUID = v.ID
		routeutils.SetItemAuthor(item, user.DisplayName, "", link)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
