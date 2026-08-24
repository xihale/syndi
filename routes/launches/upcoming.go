package routes

import (
	"fmt"
	"html"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const launchesUpcomingURL = "https://ll.thespacedevs.com/2.2.0/launch/upcoming/?limit=10&mode=list"

var launchesUpcomingRoute = routeutils.RouteSpec{
	Path:        "upcoming",
	Name:        "Upcoming Rocket Launches",
	Example:     "launches/upcoming",
	Maintainers: []string{"xihale"},
	Description: "Next upcoming rocket launches from the Launch Library 2 API (The Space Devs)",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	// The Launch Library 2 API allows only 15 requests per hour; cache aggressively.
	CacheTTL: 6 * time.Hour,
	Handler:  LaunchesUpcomingHandler,
}

// LaunchesUpcomingHandler handles /launches/upcoming
func LaunchesUpcomingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var resp llLaunchList
	if err := routeutils.GetJSON(c.Parent(), c.Client(), launchesUpcomingURL, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"Upcoming Rocket Launches",
		"https://thespacedevs.com/launches/upcoming",
		"Next upcoming rocket launches, via the Launch Library 2 API",
	)

	routeutils.AppendMappedItems(feed, resp.Results, 10, func(l llLaunch) *models.Item {
		if l.Name == "" {
			return nil
		}
		title := l.Name
		link := l.URL // mode=list has no dedicated human page; the canonical API URL is unique and stable.
		if link == "" {
			link = "https://thespacedevs.com/launches/upcoming"
		}

		desc := "<b>Provider:</b> " + html.EscapeString(l.LSPName)
		if l.Mission != "" {
			desc += "<br/><b>Mission:</b> " + html.EscapeString(l.Mission)
		}
		if l.MissionType != "" {
			desc += "<br/><b>Mission type:</b> " + html.EscapeString(l.MissionType)
		}
		pad := l.Pad
		if l.Location != "" {
			if pad != "" {
				pad += ", "
			}
			pad += l.Location
		}
		if pad != "" {
			desc += "<br/><b>Launch site:</b> " + html.EscapeString(pad)
		}
		if !l.WindowStart.IsZero() {
			window := l.WindowStart.UTC().Format(time.RFC3339)
			if !l.WindowEnd.IsZero() {
				window += " – " + l.WindowEnd.UTC().Format(time.RFC3339)
			}
			desc += "<br/><b>Launch window (UTC):</b> " + window
		}
		if l.Status.Name != "" {
			desc += "<br/><b>Status:</b> " + html.EscapeString(l.Status.Name)
		}
		if l.Image != "" {
			desc += fmt.Sprintf(`<br/><img src="%s"/>`, html.EscapeString(l.Image))
		}

		item := routeutils.NewItem(title, link, desc, l.Net)
		if l.ID != "" {
			item.GUID = "launchlibrary-" + l.ID
		}
		return item
	})
	return feed, nil
}

// llLaunchList is the simplified payload returned by ?mode=list.
type llLaunchList struct {
	Count   int        `json:"count"`
	Results []llLaunch `json:"results"`
}

type llLaunch struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status struct {
		Name string `json:"name"`
	} `json:"status"`
	Net         time.Time `json:"net"` // expected launch time
	WindowEnd   time.Time `json:"window_end"`
	WindowStart time.Time `json:"window_start"`
	LSPName     string    `json:"lsp_name"` // launch service provider
	Mission     string    `json:"mission"`
	MissionType string    `json:"mission_type"`
	Pad         string    `json:"pad"`
	Location    string    `json:"location"`
	Image       string    `json:"image"`
}
