package routes

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var wikipediaFeaturedRoute = routeutils.RouteSpec{
	Path:        "featured/:date",
	Name:        "Wikipedia Featured Content",
	Example:     "wikipedia/featured/2026-08-23",
	Maintainers: []string{"xihale"},
	Description: "Featured article, most-read articles and picture of the day from Wikipedia (English), YYYY-MM-DD format",
	Categories:  []models.Category{{Name: "study"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("date", "date in YYYY-MM-DD format (e.g. 2026-08-23)"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  WikipediaFeaturedHandler,
}

// WikipediaFeaturedHandler handles /wikipedia/featured/:date
func WikipediaFeaturedHandler(c *ctxpkg.Context) (*models.Feed, error) {
	dateParam := c.Param("date")
	date, err := time.Parse("2006-01-02", dateParam)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q, expected YYYY-MM-DD format like 2026-08-23", dateParam)
	}
	url := wikipediaAPIBase + "/featured/" + date.Format("2006/01/02")

	var resp wikiFeatured
	if err := routeutils.GetJSON(c.Parent(), c.Client(), url, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"Wikipedia Featured Content",
		wikipediaBaseURL,
		"Featured article, most-read articles and media from Wikipedia for "+dateParam,
	)

	// Today's featured article.
	if tfa := resp.TFA; len(tfa.Titles.Normalized) > 0 || tfa.Title != "" {
		if link := tfa.link(); link != "" {
			desc := wikiPageDescription(tfa)
			item := routeutils.NewItem(tfa.Titles.Normalized, link, desc, date)
			item.GUID = "wikipedia-tfa-" + dateParam
			routeutils.AddItem(feed, item)
		}
	}

	// Most read articles (top 3).
	for _, article := range resp.MostRead.Articles {
		if len(feed.Items) >= 4 {
			break
		}
		title := article.Titles.Normalized
		if title == "" {
			continue
		}
		link := article.link()
		if link == "" {
			continue
		}

		desc := wikiPageDescription(article)
		desc += "<br/><b>Views:</b> " + html.EscapeString(formatInt(article.Views))
		if article.Rank > 0 {
			desc += " (#" + strconv.Itoa(article.Rank) + " most read)"
		}

		item := routeutils.NewItem(title, link, desc, date)
		item.GUID = "wikipedia-mostread-" + dateParam + "-" + article.Titles.Canonical
		routeutils.AddItem(feed, item)
	}

	// Picture of the day.
	if img := resp.Image; img.Thumbnail.Source != "" {
		desc := fmt.Sprintf(`<img src="%s"/>`, html.EscapeString(img.Thumbnail.Source))
		if text := strings.TrimSpace(img.Desc.Text); text != "" {
			desc += "<br/>" + html.EscapeString(text)
		}
		if credit := strings.TrimSpace(img.Credit.Text); credit != "" {
			desc += "<br/>Credit: " + html.EscapeString(credit)
		}
		link := img.FilePage
		if link == "" {
			link = wikipediaBaseURL
		}
		itemTitle := img.Title
		if itemTitle == "" {
			itemTitle = "Picture of the Day"
		} else {
			itemTitle = "Picture of the Day: " + itemTitle
		}
		item := routeutils.NewItem(itemTitle, link, desc, date)
		item.GUID = "wikipedia-potd-" + dateParam
		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

// wikiPageDescription renders a page thumbnail and extract as HTML.
func wikiPageDescription(p wikiPage) string {
	desc := ""
	if p.Thumbnail.Source != "" {
		desc += fmt.Sprintf(`<img src="%s"/><br/>`, html.EscapeString(p.Thumbnail.Source))
	}
	if p.Extract != "" {
		desc += html.EscapeString(p.Extract)
	}
	return desc
}

func formatInt(n int) string {
	return strconv.FormatInt(int64(n), 10)
}

type wikiFeatured struct {
	TFA      wikiPage `json:"tfa"`
	MostRead struct {
		Date     string     `json:"date"`
		Articles []wikiPage `json:"articles"`
	} `json:"mostread"`
	Image wikiFeaturedImage `json:"image"`
}

type wikiFeaturedImage struct {
	Title     string `json:"title"`
	FilePage  string `json:"file_page"`
	Thumbnail struct {
		Source string `json:"source"`
	} `json:"thumbnail"`
	Desc struct {
		Text string `json:"text"`
	} `json:"description"`
	Credit struct {
		Text string `json:"text"`
	} `json:"credit"`
}
