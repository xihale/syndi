package routes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// ithomeZtTimeRe extracts the datetime literal from jsDateDiff('...').
var ithomeZtTimeRe = regexp.MustCompile(`'([^']*)'`)

const ithomeZtDefaultID = "xijiayi"

// ithomeZtRoute serves /ithome/zt[/:id] — a 专题 (topic) feed such as the
// 喜加一 giveaway topic.
var ithomeZtRoute = routeutils.RouteSpec{
	Path:        "zt",
	Name:        "专题",
	Example:     "ithome/zt/xijiayi",
	URL:         "https://www.ithome.com/zt/xijiayi",
	Maintainers: []string{"xihale"},
	Description: "ITHome topic feed; id 缺省为 xijiayi(喜加一), 更多专题见 ithome.com/zt",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Maximum number of items to enrich with article content, default 20, max 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  ITHomeZTHandler,
}

// ithomeZTIDRoute registers the deeper /zt/:id shape (gin has no optional
// segments).
var ithomeZTIDRoute = func() routeutils.RouteSpec {
	clone := ithomeZtRoute
	clone.Path = "zt/:id"
	clone.Parameters = append(
		[]models.Parameter{routeutils.RequiredParam("id", "专题 id, e.g. xijiayi")},
		clone.Parameters...)
	return clone
}()

// ITHomeZTHandler handles /ithome/zt[/:id].
func ITHomeZTHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	if id == "" {
		id = ithomeZtDefaultID
	}
	pageURL := fmt.Sprintf("%s/zt/%s", ithomeRoot, id)

	doc, err := ithomeProfile.Fetch(pageURL).GetHTML(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}

	var entries []ithomeListEntry
	doc.Each("div.newsbody", func(_ int, item *parser.Selection) {
		a := item.Find("a")
		if a == nil {
			return
		}
		href, ok := a.Attr("href")
		if !ok || href == "" {
			return
		}
		title := ""
		if h2 := item.Find("h2"); h2 != nil {
			title = h2.TextTrim()
		}
		if title == "" {
			title, _ = a.Attr("title")
		}
		if title == "" {
			return
		}
		entry := ithomeListEntry{Title: title, Link: href}
		if p := item.Find("p.hidden-xs"); p != nil {
			entry.Summary = p.TextTrim()
		}
		entry.PubDate = ithomeZtItemTime(item)
		if editor := item.Find(".editor"); editor != nil && editor.Length() > 0 {
			// Editor cell renders as "<name> · <stats>"; keep the name part.
			if name := strings.TrimSpace(strings.SplitN(editor.Text(), "·", 2)[0]); name != "" && !strings.ContainsAny(name, "\n评论") {
				entry.Author = name
			}
		}
		entries = append(entries, entry)
	})
	if len(entries) == 0 {
		return nil, fmt.Errorf("ithome: 专题 %s 无条目", id)
	}

	feedTitle := fmt.Sprintf("IT之家 - 专题 %s", id)
	if pageTitle := strings.TrimSpace(doc.Text("head title")); pageTitle != "" {
		feedTitle = "IT之家 - " + pageTitle
	}
	feed := routeutils.NewFeed(feedTitle, pageURL, fmt.Sprintf("IT之家%s专题动态", id))
	ithomeAppendDetailed(c, feed, "ithome-zt-", entries, parseITHomeLimit(c, 20))
	return feed, nil
}

// ithomeZtItemTime parses the embedded jsDateDiff('YYYY/M/D H:m:s') literal.
func ithomeZtItemTime(item *parser.Selection) time.Time {
	script := item.Find(".time script")
	if script == nil {
		return time.Time{}
	}
	m := ithomeZtTimeRe.FindStringSubmatch(script.Text())
	if m == nil {
		return time.Time{}
	}
	if t, err := time.ParseInLocation("2006/1/2 15:04:05", m[1], ithomeCSTZone); err == nil {
		return t
	}
	return time.Time{}
}
