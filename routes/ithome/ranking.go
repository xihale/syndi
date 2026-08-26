package routes

import (
	"fmt"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// ithomeRankKind is one ranking board on /block/rank.html.
type ithomeRankKind struct {
	Type      string // route param
	SectionID string // <ul> id on the block page
	Title     string
}

var ithomeRankKinds = []ithomeRankKind{
	{"24h", "d-1", "24 小时最热"},
	{"7days", "d-2", "7 天最热"},
	{"monthly", "d-3", "月榜"},
}

// ithomeRankingRoute serves /ithome/ranking/:type — the IT之家 hot boards
// scraped from www.ithome.com/block/rank.html (upstream 24h/7days/monthly).
var ithomeRankingRoute = routeutils.RouteSpec{
	Path:        "ranking/:type",
	Name:        "热榜",
	Example:     "ithome/ranking/24h",
	Maintainers: []string{"xihale"},
	Description: "ITHome ranking boards: 24h 24小时阅读榜, 7days 7天最热, monthly 月榜",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("type", "榜单类别: 24h / 7days / monthly"),
		routeutils.OptionalParam("limit", "Maximum number of items to enrich with article content, default all, max 50"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  ITHomeRankingHandler,
}

// ITHomeRankingHandler handles /ithome/ranking/:type.
func ITHomeRankingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	rankType := c.Param("type")
	var kind *ithomeRankKind
	for i := range ithomeRankKinds {
		if ithomeRankKinds[i].Type == rankType {
			kind = &ithomeRankKinds[i]
			break
		}
	}
	if kind == nil {
		return nil, fmt.Errorf("ithome: 无效的榜单类型 %q，支持 24h / 7days / monthly", rankType)
	}

	pageURL := ithomeRoot + "/block/rank.html"
	doc, err := ithomeProfile.Fetch(pageURL).GetHTML(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}
	section := doc.FindSelector("#" + kind.SectionID)
	if section == nil {
		return nil, fmt.Errorf("ithome: 榜单区块 #%s 未找到 (%s)", kind.SectionID, pageURL)
	}

	var entries []ithomeListEntry
	section.Find("li a").Each(func(_ int, a *parser.Selection) {
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		title := a.TextTrim()
		if title == "" {
			title, _ = a.Attr("title")
		}
		entries = append(entries, ithomeListEntry{Title: title, Link: href})
	})
	if len(entries) == 0 {
		return nil, fmt.Errorf("ithome: 榜单 %s 无条目", rankType)
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("IT之家-%s", kind.Title),
		ithomeRoot+"/",
		fmt.Sprintf("IT之家%s榜", kind.Title),
	)
	limit := parseITHomeLimit(c, 0)
	ithomeAppendDetailed(c, feed, "ithome-ranking-", entries, limit)
	return feed, nil
}
