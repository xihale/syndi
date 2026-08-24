package routes

import (
	"fmt"
	"html"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var wikipediaOnThisDayRoute = routeutils.RouteSpec{
	Path:        "onthisday/:monthday",
	Name:        "Wikipedia On This Day",
	Example:     "wikipedia/onthisday/08-24",
	Maintainers: []string{"xihale"},
	Description: "Historical events on this day from Wikipedia (English), MM-DD format",
	Categories:  []models.Category{{Name: "study"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("monthday", "month and day, MM-DD (e.g. 08-24)"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  WikipediaOnThisDayHandler,
}

// WikipediaOnThisDayHandler handles /wikipedia/onthisday/:monthday
func WikipediaOnThisDayHandler(c *ctxpkg.Context) (*models.Feed, error) {
	monthday := c.Param("monthday")
	mm, dd, err := validateMonthDay(monthday)
	if err != nil {
		return nil, err
	}
	url := wikipediaAPIBase + "/onthisday/events/" + mm + "/" + dd

	var resp wikiOnThisDay
	if err := routeutils.GetJSON(c.Parent(), c.Client(), url, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"Wikipedia On This Day",
		wikipediaBaseURL+"wiki/On_this_day_in_history",
		"Historical events that happened on "+mm+"-"+dd+", selected by Wikipedia",
	)

	routeutils.AppendMappedItems(feed, resp.Events, 0, func(e wikiEvent) *models.Item {
		if e.Text == "" || len(e.Pages) == 0 {
			return nil
		}
		link := e.Pages[0].link()
		if link == "" {
			return nil
		}

		desc := html.EscapeString(e.Text)
		if thumb := e.Pages[0].Thumbnail.Source; thumb != "" {
			desc = fmt.Sprintf(`<img src="%s"/><br/>`, html.EscapeString(thumb)) + desc
		}
		for _, page := range e.Pages[1:] {
			if pageLink := page.link(); pageLink != "" && len(page.Titles.Normalized) > 0 {
				desc += fmt.Sprintf(`<br/><a href="%s">%s</a>`, html.EscapeString(pageLink), html.EscapeString(page.Titles.Normalized))
			}
		}

		itemTitle := truncateTitle(e.Text, 120)
		item := routeutils.NewItem(itemTitle, link, desc, onThisDayEventDate(e.Year, atoi(mm), atoi(dd)))
		item.GUID = fmt.Sprintf("wikipedia-onthisday-%d-%s-%d", e.Year, mm+dd, hashString(e.Text))
		return item
	})
	return feed, nil
}

type wikiOnThisDay struct {
	Events []wikiEvent `json:"events"`
}

type wikiEvent struct {
	Text  string     `json:"text"`
	Year  int        `json:"year"`
	Pages []wikiPage `json:"pages"`
}
