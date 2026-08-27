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

// doubanGroupTypes maps the group homepage discussion filter.
var doubanGroupTypes = map[string]string{
	"essence": "最热",
	"elite":   "精华",
}

var doubanGroupRoute = routeutils.RouteSpec{
	Path:        "group/:groupid",
	Name:        "Douban Group Discussions",
	Example:     "douban/group/648102",
	Maintainers: []string{"xihale"},
	Description: "Latest topics of a douban group (豆瓣小组)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("groupid", "Douban group id, e.g. 648102"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  DoubanGroupHandler,
}

var doubanGroupTypeRoute = routeutils.RouteSpec{
	Path:        "group/:groupid/:type",
	Name:        "Douban Group Discussions by Type",
	Example:     "douban/group/648102/essence",
	Maintainers: []string{"xihale"},
	Description: "Topics of a douban group filtered by type (豆瓣小组讨论)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("groupid", "Douban group id, e.g. 648102"),
		routeutils.RequiredParam("type", "Discussion type: latest (最新, default), essence (最热), elite (精华)"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  DoubanGroupHandler,
}

const doubanGroupItemLimit = 30

// DoubanGroupHandler handles /douban/group/:groupid/:type?
func DoubanGroupHandler(c *ctxpkg.Context) (*models.Feed, error) {
	groupID := doubanSanitizeKey(c.Param("groupid"), "")
	if groupID == "" {
		return nil, fmt.Errorf("douban group: invalid groupid %q", c.Param("groupid"))
	}
	groupType := c.Param("type")
	if _, ok := doubanGroupTypes[doubanSanitizeKey(groupType, "")]; !ok {
		groupType = ""
	}

	target := fmt.Sprintf("%s/group/%s/", doubanWWWBaseURL, groupID)
	if groupType != "" {
		target += "?type=" + groupType
	}
	ctx := c.Parent()
	doc, err := doubanFetchHTML(ctx, c.Client(), target, doubanWWWBaseURL+"/")
	if err != nil {
		return nil, err
	}

	name := groupName(doc)
	title := "豆瓣小组-" + firstNonEmpty(name, groupID)
	feed := routeutils.NewFeed(title, target, title)
	doubanAppendGroupItems(feed, doc)
	return feed, nil
}

// groupName extracts the group name from the page h1.
func groupName(doc *parser.Document) string {
	return routeutils.CollapseWhitespace(doc.First("#content h1").TextTrim())
}

// doubanAppendGroupItems parses the .olt discussion table.
func doubanAppendGroupItems(feed *models.Feed, doc *parser.Document) {
	appended := 0
	doc.Each(".olt tr", func(_ int, row *parser.Selection) {
		if appended >= doubanGroupItemLimit || row.HasClass("th") {
			return
		}
		if item := parseDoubanGroupRow(row); item != nil {
			routeutils.AddItem(feed, item)
			appended++
		}
	})
}

// parseDoubanGroupRow converts one table row into an item.
func parseDoubanGroupRow(row *parser.Selection) *models.Item {
	linkSel := row.Find(".title a")
	title := linkSel.AttrOr("title", "")
	if title == "" {
		title = routeutils.CollapseWhitespace(linkSel.TextTrim())
	}
	link := linkSel.AttrOr("href", "")
	if title == "" || link == "" {
		return nil
	}

	author := routeutils.CollapseWhitespace(row.Find("td").Eq(1).Find("a").TextTrim())
	replies := strings.TrimSpace(row.Find(".r-count").TextTrim())
	lastReply := strings.TrimSpace(row.Find("td.time").TextTrim())

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("标题：%s<br>", html.EscapeString(title)))
	if author != "" {
		sb.WriteString(fmt.Sprintf("作者：%s<br>", html.EscapeString(author)))
	}
	if replies != "" {
		sb.WriteString(fmt.Sprintf("回复：%s<br>", html.EscapeString(replies)))
	}
	if lastReply != "" {
		sb.WriteString(fmt.Sprintf("最后回复：%s", html.EscapeString(lastReply)))
	}

	item := routeutils.NewItem(title, link, sb.String(), time.Time{})
	if item == nil {
		return nil
	}
	if author != "" {
		routeutils.SetAuthor(item, author)
	}
	item.GUID = "douban-group-" + firstNonEmpty(doubanIDFromLink(link), title)
	return item
}
