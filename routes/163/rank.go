package routes

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

const neteaseRootURL = "https://news.163.com"

func neteaseProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer(neteaseRootURL + "/")
}

type neteaseRankCategory struct {
	link  string
	title string
}

var neteaseRankCategories = map[string]neteaseRankCategory{
	"whole":         {link: "/special/0001386F/rank_whole.html", title: "全站"},
	"news":          {link: "/special/0001386F/rank_news.html", title: "新闻"},
	"entertainment": {link: "/special/0001386F/rank_ent.html", title: "娱乐"},
	"sports":        {link: "/special/0001386F/rank_sports.html", title: "体育"},
	"money":         {link: "https://money.163.com/special/002526BH/rank.html", title: "财经"},
	"tech":          {link: "/special/0001386F/rank_tech.html", title: "科技"},
	"auto":          {link: "/special/0001386F/rank_auto.html", title: "汽车"},
	"lady":          {link: "/special/0001386F/rank_lady.html", title: "女人"},
	"house":         {link: "/special/0001386F/rank_house.html", title: "房产"},
	"game":          {link: "/special/0001386F/game_rank.html", title: "游戏"},
	"travel":        {link: "/special/0001386F/rank_travel.html", title: "旅游"},
	"edu":           {link: "/special/0001386F/rank_edu.html", title: "教育"},
}

var neteaseRankTypes = map[string]string{"click": "点击榜", "follow": "跟帖榜"}
var neteaseRankTimeTitles = map[string]string{"hour": "1小时", "day": "24小时", "week": "本周", "month": "本月"}

const neteaseTabMarker = `class="tabContents"`

// Row markup varies by position: ranks 1-3 use class="red", 4-10 "gray"
// (both with a <span> rank), 11+ use class="rank" without a span.
var neteaseRankRowRe = regexp.MustCompile(`<td class="(?:red|gray|rank)">(?:<span>\d+</span>)?<a href="([^"]+)">([^<]+)</a></td>\s*<td class="[^"]*">([^<]*)</td>`)

var (
	neteaseOGTitleRe = regexp.MustCompile(`property="og:title" content="([^"]*)"`)
	neteaseOGDateRe  = regexp.MustCompile(`property="og:release_date" content="([^"]*)"`)
)

var neteaseRankRoute = routeutils.RouteSpec{
	Path:        "news/rank",
	Name:        "Netease News Rank",
	Example:     "163/news/rank",
	Maintainers: []string{"xihale"},
	Description: "Netease News (网易新闻) whole-site click rank of the last 24 hours",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Max items, default 20"),
	},
	CacheTTL: 60 * time.Minute,
	Handler: func(c *ctxpkg.Context) (*models.Feed, error) {
		return neteaseRankFeed(c, "whole", "click", "day")
	},
}

var neteaseRankParamsRoute = routeutils.RouteSpec{
	Path:        "news/rank/:category/:type/:time",
	Name:        "Netease News Rank By Params",
	Example:     "163/news/rank/tech/click/day",
	Maintainers: []string{"xihale"},
	Description: "Netease News (网易新闻) rank board by category/type/time. Categories: whole/news/entertainment/sports/money/tech/auto/lady/house/game/travel/edu; types: click/follow; times: day/week/month (upstream no longer publishes hourly boards; hour falls back to day)",
	Categories:  []models.Category{{Name: "new-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("category", "News category slug, e.g. tech"),
		routeutils.RequiredParam("type", "Board type: click or follow"),
		routeutils.RequiredParam("time", "Time range: day, week or month"),
		routeutils.OptionalParam("limit", "Max items, default 20"),
	},
	CacheTTL: 60 * time.Minute,
	Handler:  NeteaseRankHandler,
}

// NeteaseRankHandler handles /163/news/rank/:category/:type/:time
func NeteaseRankHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return neteaseRankFeed(c, c.Param("category"), c.Param("type"), c.Param("time"))
}

func neteaseRankFeed(c *ctxpkg.Context, category, typ, timeRange string) (*models.Feed, error) {
	if _, ok := neteaseRankTimeTitles[timeRange]; !ok {
		return nil, fmt.Errorf("invalid time %q; use day/week/month", timeRange)
	}
	if _, ok := neteaseRankTypes[typ]; !ok {
		return nil, fmt.Errorf("invalid type %q; use click/follow", typ)
	}
	cfg, ok := neteaseRankCategories[category]
	if !ok {
		return nil, fmt.Errorf("invalid category %q; see route description", category)
	}
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)

	currentURL := cfg.link
	if !strings.HasPrefix(currentURL, "http") {
		currentURL = neteaseRootURL + currentURL
	}
	body, err := neteaseProfile().Fetch(currentURL).GetString(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}
	entries := neteaseParseRows(neteaseSelectBoard(body, typ, timeRange))
	if len(entries) == 0 {
		return nil, fmt.Errorf("netease: no rank rows found at %s", currentURL)
	}

	boardName := map[string]string{"click": "点击", "follow": "跟帖"}[typ]
	feed := routeutils.NewFeed(
		fmt.Sprintf("网易新闻%s%s榜 - %s", neteaseRankTimeTitles[timeRange], boardName, cfg.title),
		currentURL,
		fmt.Sprintf("网易新闻%s%s榜（%s）", neteaseRankTimeTitles[timeRange], boardName, cfg.title),
	)

	n := 0
	for _, e := range entries {
		if n >= limit {
			break
		}
		desc := ""
		title := e.title
		var pubDate time.Time
		if d, t, pd := neteaseFetchArticle(c, e.url); d != "" {
			desc = d
			pubDate = pd
			if t != "" {
				title = t
			}
		} else if e.count != "" {
			desc = "<p>" + html.EscapeString("热度："+e.count) + "</p>"
		}
		item := routeutils.NewItem(title, e.url, desc, pubDate)
		if item == nil {
			continue
		}
		routeutils.AddItem(feed, item)
		n++
	}
	return feed, nil
}

type neteaseRankEntry struct {
	url   string
	title string
	count string
}

// neteaseSelectBoards flattens every <table> of every tabContents block into an
// ordered list of boards. Observed page layout (uniform across categories):
// board0 = 24h clicks, board1 = week clicks, board2 = month clicks,
// board3 = today follows, board4+ = week/month follows.
func neteaseSelectBoards(pageHTML string) []string {
	var boards []string
	for _, block := range strings.Split(pageHTML, neteaseTabMarker)[1:] {
		for _, seg := range strings.Split(block, "<table")[1:] {
			if end := strings.Index(seg, "</table>"); end >= 0 {
				seg = seg[:end]
			}
			boards = append(boards, seg)
		}
	}
	return boards
}

// neteaseSelectBoard picks the board matching type/time from the flattened list.
func neteaseSelectBoard(pageHTML, typ, timeRange string) string {
	boards := neteaseSelectBoards(pageHTML)
	idx := map[string]int{"day": 0, "hour": 0, "week": 1, "month": 2}[timeRange]
	if typ == "follow" {
		idx += 3
	}
	if idx >= len(boards) {
		idx = len(boards) - 1
	}
	if idx < 0 {
		return ""
	}
	return boards[idx]
}

// neteaseParseRows extracts ranked entries from one board table.
func neteaseParseRows(board string) []neteaseRankEntry {
	matches := neteaseRankRowRe.FindAllStringSubmatch(board, -1)
	entries := make([]neteaseRankEntry, 0, len(matches))
	for _, m := range matches {
		entries = append(entries, neteaseRankEntry{url: m[1], title: html.UnescapeString(m[2]), count: strings.TrimSpace(m[3])})
	}
	return entries
}

var neteaseOGTitleFallbackSuffix = "_手机网易网"

// neteaseFetchArticle fetches the mobile article page for full text and date.
func neteaseFetchArticle(c *ctxpkg.Context, link string) (desc, title string, pubDate time.Time) {
	path := link
	if i := strings.Index(path, "://"); i >= 0 {
		path = path[i+3:]
		if j := strings.Index(path, "/"); j >= 0 {
			path = path[j:]
		} else {
			return "", "", time.Time{}
		}
	}
	mobileURL := "https://m.163.com" + path
	body, err := neteaseProfile().Referer(mobileURL).Fetch(mobileURL).GetString(c.Parent(), c.Client())
	if err != nil {
		return "", "", time.Time{}
	}
	if m := neteaseOGTitleRe.FindStringSubmatch(body); m != nil {
		title = strings.TrimSuffix(html.UnescapeString(m[1]), neteaseOGTitleFallbackSuffix)
	}
	if m := neteaseOGDateRe.FindStringSubmatch(body); m != nil {
		if t, err := dateutil.ParseDate(m[1]); err == nil {
			pubDate = t
		}
	}
	return neteaseExtractArticleBody(body), title, pubDate
}

var (
	neteaseBodyStartRe = regexp.MustCompile(`(?s)<section class="article-body[^"]*"[^>]*>`)
	neteaseLazyImgRe   = regexp.MustCompile(`\s(?:data-src|data-lazyload)="[^"]*"`)
)

func neteaseExtractArticleBody(pageHTML string) string {
	loc := neteaseBodyStartRe.FindStringIndex(pageHTML)
	if loc == nil {
		return ""
	}
	start := loc[1]
	end := strings.Index(pageHTML[start:], "</section>")
	if end < 0 {
		return ""
	}
	content := pageHTML[start : start+end]
	content = neteaseLazyImgRe.ReplaceAllString(content, "")
	content = strings.ReplaceAll(content, "<p></p>", "")
	content = strings.TrimSpace(content)
	if len(content) < 40 {
		return ""
	}
	return content
}
