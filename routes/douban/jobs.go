package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const doubanJobsURL = "https://jobs.douban.com/jobs"

// doubanJobsTypes maps the jobs page path segments.
var doubanJobsTypes = map[string]string{
	"social": "社会招聘",
	"campus": "校园招聘",
	"intern": "实习生招聘",
}

var doubanJobsRoute = routeutils.RouteSpec{
	Path:        "jobs/:type",
	Name:        "Douban Jobs",
	Example:     "douban/jobs/social",
	Maintainers: []string{"xihale"},
	Description: "Job openings at douban (豆瓣招聘)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("type", "Recruiting type: social (社会招聘), campus (校园招聘), intern (实习生招聘)"),
	},
	CacheTTL: 12 * time.Hour,
	Handler:  DoubanJobsHandler,
}

// DoubanJobsHandler handles /douban/jobs/:type
func DoubanJobsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	typ := routeutils.ParseEnum(c.Param("type"), "social", doubanCollectionKeys(doubanJobsTypes)...)
	name := doubanJobsTypes[typ]

	target := fmt.Sprintf("%s/%s/", doubanJobsURL, typ)
	ctx := c.Parent()
	doc, err := doubanFetchHTML(ctx, c.Client(), target, target)
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf("豆瓣%s", name)
	feed := routeutils.NewFeed(title, target, title)
	doubanAppendJobItems(feed, doc, typ)
	return feed, nil
}

func doubanAppendJobItems(feed *models.Feed, doc *parser.Document, typ string) {
	doc.Each("div.mod.position", func(_ int, sel *parser.Selection) {
		if item := parseDoubanJobItem(sel, typ); item != nil {
			routeutils.AddItem(feed, item)
		}
	})
}

// parseDoubanJobItem renders one div.mod.position block; descriptions are
// rebuilt from the em/pre section pairs to keep content XSS-safe.
func parseDoubanJobItem(sel *parser.Selection, typ string) *models.Item {
	position := routeutils.CollapseWhitespace(sel.Find("h3").TextTrim())
	if position == "" {
		return nil
	}
	anchor := sel.Find("h3").AttrOr("id", "")

	var sb strings.Builder
	sel.Find("div.bd").Each(func(_ int, bd *parser.Selection) {
		bd.Children().Each(func(_ int, child *parser.Selection) {
			text := strings.TrimSpace(child.Text())
			switch {
			case child.Is("em"):
				sb.WriteString("<strong>" + html.EscapeString(text) + "</strong><br>")
			default:
				sb.WriteString(html.EscapeString(text) + "<br>")
			}
			sb.WriteString("<br>")
		})
	})

	link := fmt.Sprintf("%s/%s/", doubanJobsURL, typ)
	if anchor != "" {
		link += "#" + anchor
	}
	item := routeutils.NewItem(position, link, sb.String(), time.Time{})
	if item == nil {
		return nil
	}
	item.GUID = "douban-jobs-" + firstNonEmpty(anchor, position)
	return item
}
