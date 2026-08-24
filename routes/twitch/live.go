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

var twitchLiveRoute = routeutils.RouteSpec{
	Path:        "live/:login",
	Name:        "Live Status",
	Example:     "twitch/live/riotgames",
	Maintainers: []string{"xihale"},
	Description: "Current live status of a Twitch channel, with one item while streaming",
	Categories:  []models.Category{{Name: "live"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("login", "Twitch username"),
	},
	CacheTTL: 5 * time.Minute,
	Handler:  TwitchLiveHandler,
}

// TwitchLiveHandler handles /twitch/live/:login.
func TwitchLiveHandler(c *ctxpkg.Context) (*models.Feed, error) {
	login := strings.ToLower(c.Param("login"))
	user, err := twitchQuery(c, gqlCall{
		Query: `query ChannelLive($login: String!) {
			user(login: $login) {
				id login displayName description profileImageURL(width: 70)
				stream { id title createdAt viewersCount game { displayName } }
			}
		}`,
		Variables: map[string]any{"login": login},
	})
	if err != nil {
		return nil, err
	}
	user, err = twitchRequireUser(user, login)
	if err != nil {
		return nil, err
	}

	link := twitchLoginLink(login)
	feed := routeutils.NewFeed(
		fmt.Sprintf("Twitch - %s - Live", user.DisplayName),
		link,
		html.EscapeString(user.Description),
	)
	if user.Stream != nil && user.Stream.ID != "" {
		title := user.Stream.Title
		if title == "" {
			title = user.DisplayName + " is live"
		}
		desc := fmt.Sprintf(`<img style="max-width:100%%;" src="https://static-cdn.jtvnw.net/previews-ttv/live_user_%s.jpg">`, login)
		if g := user.Stream.Game; g != nil && g.DisplayName != "" {
			desc += "<br>Category: " + html.EscapeString(g.DisplayName)
		}
		if user.Stream.ViewersCount > 0 {
			desc += fmt.Sprintf("<br>Viewers: %d", user.Stream.ViewersCount)
		}
		item := routeutils.NewItem(title, link, desc, user.Stream.CreatedAt)
		item.GUID = user.Stream.ID
		routeutils.SetItemAuthor(item, user.DisplayName, "", link)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
