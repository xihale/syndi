// Package routes implements RSSHub-style routes for xkcd.
package routes

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var xkcdLatestRoute = routeutils.RouteSpec{
	Path:        "latest",
	Name:        "xkcd Latest Comics",
	Example:     "xkcd/latest",
	Maintainers: []string{"xihale"},
	Description: "Latest xkcd comics",
	Categories:  []models.Category{{Name: "picture"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Number of comics (default 10, max 50)"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  XKCDLatestHandler,
}

type xkcdComic struct {
	Num   int    `json:"num"`
	Title string `json:"safe_title"`
	Alt   string `json:"alt"`
	Image string `json:"img"`
	Day   string `json:"day"`
	Month string `json:"month"`
	Year  string `json:"year"`
}

// XKCDLatestHandler handles /xkcd/latest
func XKCDLatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 10, 50)

	ctx, cancel := context.WithTimeout(c.Parent(), 60*time.Second)
	defer cancel()

	var latest xkcdComic
	if err := routeutils.GetJSON(ctx, c.Client(), "https://xkcd.com/info.0.json", &latest); err != nil {
		return nil, err
	}
	if latest.Num == 0 {
		return nil, fmt.Errorf("xkcd: unexpected latest payload")
	}

	feed := routeutils.NewFeed(
		"xkcd",
		"https://xkcd.com/",
		"xkcd: A webcomic of romance, sarcasm, math, and language.",
	)

	for i := 0; i < limit; i++ {
		comic := latest
		if i > 0 {
			if err := routeutils.GetJSON(ctx, c.Client(),
				fmt.Sprintf("https://xkcd.com/%d/info.0.json", latest.Num-i), &comic); err != nil {
				break // stop at the first unavailable older comic
			}
		}
		routeutils.AddItem(feed, newXKCDItem(comic))
	}

	return feed, nil
}

func newXKCDItem(comic xkcdComic) *models.Item {
	if comic.Num == 0 || comic.Image == "" {
		return nil
	}
	link := fmt.Sprintf("https://xkcd.com/%d/", comic.Num)

	var b strings.Builder
	fmt.Fprintf(&b, `<img src="%s" alt="%s"/>`, html.EscapeString(comic.Image), html.EscapeString(comic.Alt))
	if comic.Alt != "" {
		b.WriteString("<p>" + html.EscapeString(comic.Alt) + "</p>")
	}

	title := comic.Title
	if title == "" {
		title = "#" + strconv.Itoa(comic.Num)
	}
	item := routeutils.NewItem(title, link, b.String(), xkcdPubDate(comic))
	item.GUID = strconv.Itoa(comic.Num)
	return item
}

// xkcdPubDate builds the publish date from the upstream year/month/day fields.
func xkcdPubDate(comic xkcdComic) time.Time {
	if comic.Year == "" || comic.Month == "" || comic.Day == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", comic.Year+"-"+pad2(comic.Month)+"-"+pad2(comic.Day))
	if err != nil {
		return time.Time{}
	}
	return t
}

func pad2(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 1 {
		return "0" + s
	}
	return s
}
