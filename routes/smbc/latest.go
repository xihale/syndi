// Package routes implements RSSHub-style routes for SMBC.
package routes

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var smbcLatestRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Saturday Morning Breakfast Cereal",
	Example:     "smbc",
	Maintainers: []string{"xihale"},
	Description: "Latest Saturday Morning Breakfast Cereal comics",
	Categories:  []models.Category{{Name: "picture"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Number of comics to walk back (default 5, max 10)"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  SmbcLatestHandler,
}

// The official /rss endpoint currently serves an HTML page, so comics are
// scraped from the ComicControl pages. Each page carries one comic
// (img#cc-comic) and a rel=prev link to walk back through the archive.
// Comic image filenames embed the publish date (…-YYYYMMDD.png).

var smbcDateRe = regexp.MustCompile(`-(20\d{6})\.\w+$`)

type smbcComic struct {
	Title string
	Image string
	Hover string
	Date  time.Time
	Link  string
}

// SmbcLatestHandler handles /smbc
func SmbcLatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 5, 10)

	ctx, cancel := context.WithTimeout(c.Parent(), 60*time.Second)
	defer cancel()

	feed := routeutils.NewFeed(
		"Saturday Morning Breakfast Cereal",
		"https://www.smbc-comics.com/",
		"Latest SMBC comics by Zach Weinersmith",
	)

	link := "https://www.smbc-comics.com/"
	for i := 0; i < limit; i++ {
		doc, err := routeutils.GetHTML(ctx, c.Client(), link)
		if err != nil {
			if i == 0 {
				return nil, err
			}
			break // tolerate a broken older page
		}
		comic := parseSmbcPage(doc, link)
		if comic == nil {
			break
		}
		routeutils.AddItem(feed, newSmbcItem(comic))

		prev, _ := doc.Find("a.cc-prev").First().Attr("href")
		prev = strings.TrimSpace(prev)
		if prev == "" || prev == link {
			break
		}
		link = prev
	}

	return feed, nil
}

func parseSmbcPage(doc *parser.Document, link string) *smbcComic {
	if doc == nil {
		return nil
	}
	img := doc.Find("img#cc-comic").First()
	src := strings.TrimSpace(img.AttrOr("src", ""))
	if src == "" {
		return nil
	}
	comic := &smbcComic{
		Image: src,
		Hover: strings.TrimSpace(img.AttrOr("title", "")),
		Link:  link,
	}

	// Page title looks like "Saturday Morning Breakfast Cereal - Slug".
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if idx := strings.LastIndex(title, " - "); idx >= 0 {
		title = strings.TrimSpace(title[idx+3:])
	}
	comic.Title = title

	if m := smbcDateRe.FindStringSubmatch(src); m != nil {
		if t, err := time.ParseInLocation("20060102", m[1], time.UTC); err == nil {
			comic.Date = t
		}
	}
	return comic
}

func newSmbcItem(comic *smbcComic) *models.Item {
	var b strings.Builder
	fmt.Fprintf(&b, `<img src="%s" alt="%s"/>`, html.EscapeString(comic.Image), html.EscapeString(comic.Hover))
	if comic.Hover != "" {
		b.WriteString("<p>" + html.EscapeString(comic.Hover) + "</p>")
	}

	title := comic.Title
	if title == "" {
		if !comic.Date.IsZero() {
			title = comic.Date.Format("SMBC 2006-01-02")
		} else {
			title = "SMBC"
		}
	}

	item := routeutils.NewItem(title, comic.Link, b.String(), comic.Date)
	item.GUID = comic.Link
	return item
}
