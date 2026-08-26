package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

// ithomeTagRoute serves /ithome/tag/:name — the news list of a tag page
// (www.ithome.com/tag/<name>, which the site canonicalizes to /tags/<name>).
var ithomeTagRoute = routeutils.RouteSpec{
	Path:        "tag/:name",
	Name:        "标签",
	Example:     "ithome/tag/win11",
	URL:         "https://www.ithome.com/tag/win11",
	Maintainers: []string{"xihale"},
	Description: "ITHome tag news; 标签名可从 ithome.com/tag/<名称> 链接中获取",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("name", "标签名称, 可从网址链接中获取"),
		routeutils.OptionalParam("limit", "Maximum number of items to enrich with article content, default 20, max 50"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  ITHomeTagHandler,
}

// ITHomeTagHandler handles /ithome/tag/:name.
func ITHomeTagHandler(c *ctxpkg.Context) (*models.Feed, error) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		return nil, fmt.Errorf("ithome: 标签名称不能为空")
	}
	pageURL := fmt.Sprintf("%s/tag/%s", ithomeRoot, name)

	doc, err := ithomeProfile.Fetch(pageURL).GetHTML(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}
	if doc.Find("ul.bl").Length() == 0 {
		return nil, fmt.Errorf("ithome: 标签 %s 的列表未找到（可能不存在该标签）", name)
	}

	var entries []ithomeListEntry
	doc.Each("ul.bl > li", func(_ int, li *parser.Selection) {
		a := li.Find("h2 a")
		if a == nil {
			return
		}
		href, ok := a.Attr("href")
		if !ok || href == "" {
			return
		}
		entry := ithomeListEntry{
			Title: a.TextTrim(),
			Link:  href,
		}
		if m := li.Find(".m"); m != nil {
			entry.Summary = m.TextTrim()
		}
		// data-ot is an ISO8601 timestamp with +08:00 offset.
		if cs := li.Find(".c"); cs != nil && cs.Length() > 0 {
			if ot, ok := cs.Attr("data-ot"); ok && ot != "" {
				if t, perr := time.Parse(time.RFC3339, ot); perr == nil {
					entry.PubDate = t
				} else if t, perr := dateutil.ParseDateInLocation(ot, ithomeCSTZone); perr == nil {
					entry.PubDate = t
				}
			}
		}
		entries = append(entries, entry)
	})
	if len(entries) == 0 {
		return nil, fmt.Errorf("ithome: 标签 %s 无条目", name)
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("IT之家 - %s标签", name),
		pageURL,
		fmt.Sprintf("IT之家 %s 标签新闻", name),
	)
	ithomeAppendDetailed(c, feed, "ithome-tag-", entries, parseITHomeLimit(c, 20))
	return feed, nil
}
