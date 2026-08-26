package routes

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var v2exTabRoute = routeutils.RouteSpec{
	Path:        "tab/:tabid",
	Name:        "V2EX Tab",
	Example:     "v2ex/tab/daily",
	Maintainers: []string{"xihale"},
	Description: "Topics on a V2EX navigation tab (all, daily, tech, creative, jobs, ...)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("tabid", "Tab ID from the V2EX navigation bar, e.g. daily"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  V2EXTabHandler,
}

// v2exTabTimeLayout is the absolute timestamp rendered into the title
// attribute of tab pages (e.g. `2026-08-26 23:36:51 +08:00`).
const v2exTabTimeLayout = "2006-01-02 15:04:05 -07:00"

// V2EXTabHandler handles /v2ex/tab/:tabid.
// The old /tab/:tabid path now returns 404 upstream; the live page moved to
// /?tab=:tabid and keeps the same item markup, which we parse directly
// instead of fetching every topic page like the upstream implementation.
func V2EXTabHandler(c *ctxpkg.Context) (*models.Feed, error) {
	tabid := strings.TrimSpace(c.Param("tabid"))
	if !validV2EXTabID(tabid) {
		return nil, fmt.Errorf("invalid tab id %q", c.Param("tabid"))
	}
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 40, 100)
	ctx := c.Parent()

	pageURL := fmt.Sprintf("%s/?tab=%s", v2exBaseURL, tabid)

	doc, err := routeutils.GetHTML(ctx, c.Client(), pageURL)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("V2EX Tab: %s", tabid),
		pageURL,
		fmt.Sprintf("Latest topics on the %q tab of V2EX", tabid),
	)

	appended := 0
	doc.Each("div.cell.item", func(_ int, sel *parser.Selection) {
		if appended >= limit {
			return
		}
		item := parseV2EXTabItem(sel)
		if item == nil {
			return
		}
		routeutils.AddItem(feed, item)
		appended++
	})

	return feed, nil
}

// validV2EXTabID guards against URL injection through the tab name.
func validV2EXTabID(tabid string) bool {
	if tabid == "" || len(tabid) > 32 {
		return false
	}
	for _, r := range tabid {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func parseV2EXTabItem(sel *parser.Selection) *models.Item {
	if sel == nil || sel.Selection == nil {
		return nil
	}

	titleSel := sel.Find("span.item_title > a").First()
	href, ok := titleSel.Attr("href")
	title := strings.TrimSpace(titleSel.Text())
	if !ok || href == "" || title == "" {
		return nil
	}

	topicID := v2exTopicIDFromHref(href)
	if topicID == 0 {
		return nil // topic pages are always numeric paths
	}
	path, _, _ := strings.Cut(href, "#")
	item := routeutils.NewItem(title, v2exBaseURL+path, "", time.Time{})
	item.GUID = fmt.Sprintf("v2ex-topic-%d", topicID)

	node := sel.Find(".topic_info a.node").First()
	if nodeName := strings.TrimSpace(node.Text()); nodeName != "" {
		routeutils.SetCategories(item, nodeName)
	}

	if authorHref, exists := sel.Find(".topic_info strong > a[href*='/member/']").First().Attr("href"); exists {
		name := strings.TrimPrefix(strings.TrimSpace(authorHref), "/member/")
		if name != "" {
			routeutils.SetAuthor(item, name, routeutils.WithAuthorURI(v2exBaseURL+"/u/"+name))
		}
	}

	if ts, exists := sel.Find(".topic_info span[title]").First().Attr("title"); exists {
		if parsed, err := time.Parse(v2exTabTimeLayout, strings.TrimSpace(ts)); err == nil {
			item.PubDate = parsed
		}
	}

	return item
}

// v2exTopicIDFromHref extracts the numeric id of an /t/{id}[-fragment] link.
func v2exTopicIDFromHref(href string) int {
	href = strings.TrimPrefix(href, "/t/")
	if i := strings.IndexByte(href, '#'); i >= 0 {
		href = href[:i]
	}
	id, err := strconv.Atoi(href)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}
