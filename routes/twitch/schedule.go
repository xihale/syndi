package routes

import (
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var twitchScheduleRoute = routeutils.RouteSpec{
	Path:        "schedule/:login",
	Name:        "Stream Schedule",
	Example:     "twitch/schedule/northernlion",
	Maintainers: []string{"xihale"},
	Description: "Upcoming scheduled streams of a Twitch channel",
	Categories:  []models.Category{{Name: "live"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("login", "Twitch username"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  TwitchScheduleHandler,
}

// TwitchScheduleHandler handles /twitch/schedule/:login.
func TwitchScheduleHandler(c *ctxpkg.Context) (*models.Feed, error) {
	login := strings.ToLower(c.Param("login"))
	user, err := twitchQuery(c, gqlCall{
		Query: `query UserSchedule($login: String!) {
			user(login: $login) {
				id displayName profileImageURL(width: 70)
				channel { schedule { segments { id title startAt endAt categories { name } } } }
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
		"Twitch - "+user.DisplayName+" - Schedule",
		link,
		"Upcoming stream schedule for "+user.DisplayName,
	)
	if user.Channel == nil || user.Channel.Schedule == nil {
		return feed, nil
	}
	for _, seg := range user.Channel.Schedule.Segments {
		if seg.Title == "" {
			continue
		}
		cats := make([]string, 0, len(seg.Categories))
		for _, cat := range seg.Categories {
			if cat.Name != "" {
				cats = append(cats, cat.Name)
			}
		}
		desc := "Start: " + seg.StartAt.UTC().Format(time.RFC3339) +
			"<br>End: " + seg.EndAt.UTC().Format(time.RFC3339)
		if len(cats) > 0 {
			desc += "<br>Categories: " + html.EscapeString(strings.Join(cats, ", "))
		}
		item := routeutils.NewItem(seg.Title, link, desc, seg.StartAt)
		if seg.ID != "" {
			item.GUID = seg.ID
		}
		routeutils.SetItemAuthor(item, user.DisplayName, "", link)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
