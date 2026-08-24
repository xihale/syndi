package routes

import (
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

// DEMO_KEY is NASA's officially documented public demo key.
const nasaAPODURL = "https://api.nasa.gov/planetary/apod?api_key=DEMO_KEY&thumbs=false&count=10"

var nasaAPODRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "NASA Astronomy Picture of the Day",
	Example:     "nasa-apod",
	Maintainers: []string{"xihale"},
	Description: "Astronomy Picture of the Day entries from NASA APOD",
	Categories:  []models.Category{{Name: "picture"}},
	Features:    models.Features{SupportRadar: true},
	// api.nasa.gov rate limits the shared DEMO_KEY aggressively; cache for a long time.
	CacheTTL: 12 * time.Hour,
	Handler:  NASAAPODHandler,
}

// NASAAPODHandler handles /nasa-apod
func NASAAPODHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var entries []apodEntry
	if err := routeutils.GetJSON(c.Parent(), c.Client(), nasaAPODURL, &entries); err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date > entries[j].Date
	})

	feed := routeutils.NewFeed(
		"NASA Astronomy Picture of the Day",
		"https://apod.nasa.gov/apod/",
		"Astronomy Picture of the Day, discover the cosmos via NASA and Michigan Technological University",
	)

	routeutils.AppendMappedItems(feed, entries, 0, func(e apodEntry) *models.Item {
		if e.MediaType != "image" || e.Title == "" || e.Date == "" {
			return nil
		}
		link := fmt.Sprintf("https://apod.nasa.gov/apod/ap%s.html", apodCompactDate(e.Date))

		desc := fmt.Sprintf(`<img src="%s"/><br/>`, html.EscapeString(e.URL))
		if e.Explanation != "" {
			desc += "<p>" + html.EscapeString(e.Explanation) + "</p>"
		}
		if e.HDURL != "" {
			desc += fmt.Sprintf(`<br/><a href="%s">HD image</a>`, html.EscapeString(e.HDURL))
		}
		if strings.TrimSpace(e.Copyright) != "" {
			desc += "<br/>Copyright: " + html.EscapeString(strings.TrimSpace(e.Copyright))
		}

		pubDate := time.Time{}
		if parsed, err := dateutil.ParseDate(e.Date); err == nil {
			pubDate = parsed
		}
		item := routeutils.NewItem(e.Title, link, desc, pubDate)
		item.GUID = "apod-" + e.Date
		return item
	})
	return feed, nil
}

// apodCompactDate converts 2008-04-20 to the APOD page code 080420 (apYYMMDD).
func apodCompactDate(date string) string {
	parts := strings.Split(date, "-")
	if len(parts) != 3 {
		return ""
	}
	year, month, day := parts[0], parts[1], parts[2]
	if len(year) == 4 {
		year = year[2:]
	}
	if len(year) != 2 || len(month) != 2 || len(day) != 2 {
		return ""
	}
	return year + month + day
}

type apodEntry struct {
	Date        string `json:"date"`
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	URL         string `json:"url"`
	HDURL       string `json:"hdurl"`
	MediaType   string `json:"media_type"`
	Copyright   string `json:"copyright"`
}
