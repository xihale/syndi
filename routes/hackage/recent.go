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

var hackageRecentRoute = routeutils.RouteSpec{
	Path:        "recent",
	Name:        "Hackage Recent Packages",
	Example:     "hackage/recent",
	Maintainers: []string{"xihale"},
	Description: "Recently uploaded or updated packages on Hackage",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  []models.Parameter{},
	CacheTTL:    30 * time.Minute,
	Handler:     HackageRecentHandler,
}

// hackageEntryRe splits a package link text like "effectful-2.7.1.0".
var hackageEntryRe = regexp.MustCompile(`^(.+)-([0-9]+(?:\.[0-9]+)+)$`)

// HackageRecentHandler handles /hackage/recent
func HackageRecentHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()

	url := "https://hackage.haskell.org/packages/recent"
	doc, err := routeutils.GetHTML(ctx, c.Client(), url)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"Hackage Recent Packages",
		url,
		"Recently uploaded or updated packages on Hackage",
	)

	count := 0
	doc.Each("table tr", func(_ int, row *parser.Selection) {
		if count >= 30 || row == nil {
			return
		}
		item := parseHackageRow(row)
		routeutils.AddItem(feed, item)
		if item != nil {
			count++
		}
	})

	return feed, nil
}

func parseHackageRow(row *parser.Selection) *models.Item {
	var cells []string
	linkHref, uploader := "", ""
	row.Find("td").Each(func(i int, cell *parser.Selection) {
		cells = append(cells, strings.TrimSpace(cell.Text()))
		if i == 1 {
			uploader = cells[1]
		}
		cell.Find("a[href^='/package/']").Each(func(_ int, a *parser.Selection) {
			href, ok := a.Attr("href")
			if ok && linkHref == "" {
				linkHref = href
			}
		})
	})
	if len(cells) < 3 || linkHref == "" {
		return nil
	}

	name, version := splitHackageName(strings.TrimSpace(cells[2]))
	title := strings.TrimSpace(cells[2])

	var published time.Time
	if t, err := time.Parse(time.RFC3339, cells[0]); err == nil {
		published = t
	}

	link := "https://hackage.haskell.org" + linkHref
	description := ""
	if uploader != "" {
		description = fmt.Sprintf("Uploaded by %s", html.EscapeString(uploader))
	}

	item := routeutils.NewItem(title, link, description, published)
	item.GUID = link
	routeutils.SetCategories(item, name)
	if version != "" {
		routeutils.SetCategories(item, version)
	}
	if uploader != "" {
		routeutils.SetItemAuthor(item, uploader, "", "")
	}
	return item
}

// splitHackageName splits "effectful-2.7.1.0" into name and version.
func splitHackageName(text string) (string, string) {
	if m := hackageEntryRe.FindStringSubmatch(text); m != nil {
		return m[1], m[2]
	}
	return text, ""
}
