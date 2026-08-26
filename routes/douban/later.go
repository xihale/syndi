package routes

import (
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

var doubanMovieLaterRoute = routeutils.RouteSpec{
	Path:        "movie/later",
	Name:        "Upcoming Movies in Cinemas",
	Example:     "douban/movie/later",
	Maintainers: []string{"xihale"},
	Description: "Movies screening soon (即将上映的电影), from the cinema upcoming page",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    6 * time.Hour,
	Handler:     DoubanMovieLaterHandler,
}

// DoubanMovieLaterHandler handles /douban/movie/later
func DoubanMovieLaterHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()

	doc, err := doubanFetchHTML(ctx, c.Client(), doubanBaseURL+"/cinema/later/beijing/", doubanBaseURL+"/")
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"即将上映的电影",
		doubanBaseURL+"/cinema/later/",
		"即将上映的电影",
	)
	doubanAppendLaterItems(feed, doc)
	return feed, nil
}

// doubanAppendLaterItems parses the upcoming list of the cinema page.
func doubanAppendLaterItems(feed *models.Feed, doc *parser.Document) {
	doc.Each("#showing-soon .item", func(_ int, sel *parser.Selection) {
		if item := parseDoubanLaterItem(sel); item != nil {
			routeutils.AddItem(feed, item)
		}
	})
}

// doubanMonthDayPattern matches date lines like "08月28日".
var doubanMonthDayPattern = regexp.MustCompile(`^\d{1,2}月\d{1,2}日$`)

// parseDoubanLaterItem extracts one #showing-soon .item entry.
func parseDoubanLaterItem(sel *parser.Selection) *models.Item {
	nameSel := sel.Find("h3 a")
	title := routeutils.CollapseWhitespace(nameSel.TextTrim())
	link := nameSel.AttrOr("href", "")
	if link == "" {
		link = sel.Find("a.thumb").AttrOr("href", "")
	}
	if title == "" || link == "" {
		return nil
	}

	lines := make([]string, 0, 4)
	sel.Find("ul li").Each(func(_ int, li *parser.Selection) {
		if text := routeutils.CollapseWhitespace(li.TextTrim()); text != "" {
			lines = append(lines, text)
		}
	})
	date := ""
	genre := ""
	wantCount := ""
	for _, line := range lines {
		switch {
		case date == "" && doubanMonthDayPattern.MatchString(line):
			date = line
		case genre == "":
			genre = line
		default:
			if strings.HasSuffix(line, "人想看") {
				wantCount = strings.TrimSuffix(line, "人想看")
			}
		}
	}

	upcomingTitle := fmt.Sprintf("%s - 《%s》 - %s", html.EscapeString(date), title, html.EscapeString(genre))

	var sb strings.Builder
	if poster := sel.Find("a.thumb img").AttrOr("src", ""); poster != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s" alt="%s"/><br>`, html.EscapeString(poster), html.EscapeString(title)))
	}
	sb.WriteString(fmt.Sprintf("标题：《%s》", html.EscapeString(title)))
	if len(lines) > 0 {
		sb.WriteString("<br>" + html.EscapeString(strings.Join(lines, " / ")))
	}
	if wantCount != "" {
		sb.WriteString(fmt.Sprintf("<br>想看人数：%s", html.EscapeString(wantCount)))
	}

	item := routeutils.NewItem(upcomingTitle, link, sb.String(), time.Time{})
	if item == nil {
		return nil
	}
	item.GUID = "douban-later-" + firstNonEmpty(doubanIDFromLink(link), title)
	return item
}
