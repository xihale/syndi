package routes

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var (
	peopleDateRe = regexp.MustCompile(`(\d{4})年(\d{2})月(\d{2})日\s*(\d{2}):(\d{2})`)
	peopleSiteRe = regexp.MustCompile(`^[a-z0-9]+$`)
	peopleListRe = regexp.MustCompile(`(?s)<a href="(https?://[^"]+\.html)"[^>]*>([^<]{6,120})</a>\s*\[(\d{4}年\d{2}月\d{2}日\s*\d{2}:\d{2})\]`)
)

func peopleProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9")
}

type peopleEntry struct {
	link  string
	title string
	date  string
}

var peopleHeadlinesRoute = routeutils.RouteSpec{
	Path:        "headlines",
	Name:        "People Headlines",
	Example:     "people/headlines",
	Maintainers: []string{"xihale"},
	Description: "People's Daily Online (人民网) homepage headlines",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Max items, default 20"),
	},
	CacheTTL: 15 * time.Minute,
	Handler: func(c *ctxpkg.Context) (*models.Feed, error) {
		return peopleChannelFeed(c, "www", "59476", "人民网-要闻")
	},
}

var peopleChannelRoute = routeutils.RouteSpec{
	Path:        "channel/:site/:category",
	Name:        "People Channel",
	Example:     "people/channel/www/59476",
	Maintainers: []string{"xihale"},
	Description: "People.com.cn (人民网) channel feed, e.g. site www category 59476 (要闻), cpc 64093. Category id comes from the channel URL path /GB/<id>/",
	Categories:  []models.Category{{Name: "new-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("site", "Subdomain, e.g. www, society, world, cpc"),
		routeutils.RequiredParam("category", "Category id from the /GB/<id>/ URL path"),
		routeutils.OptionalParam("limit", "Max items, default 20"),
	},
	CacheTTL: 20 * time.Minute,
	Handler:  PeopleChannelHandler,
}

// PeopleChannelHandler handles /people/channel/:site/:category
func PeopleChannelHandler(c *ctxpkg.Context) (*models.Feed, error) {
	site := c.Param("site")
	category := c.Param("category")
	if !peopleSiteRe.MatchString(site) || !peopleSiteRe.MatchString(category) {
		return nil, fmt.Errorf("invalid site/category; expected alphanumeric values")
	}
	return peopleChannelFeed(c, site, category, fmt.Sprintf("人民网-%s/%s", site, category))
}

func peopleChannelFeed(c *ctxpkg.Context, site, category, feedTitle string) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	pageURL := fmt.Sprintf("http://%s.people.com.cn/GB/%s/index.html", site, category)

	body, err := peopleProfile().Referer(pageURL).Fetch(pageURL).GetString(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(feedTitle, pageURL, feedTitle+"最新内容")
	seen := make(map[string]bool, 64)
	n := 0
	for _, m := range peopleListRe.FindAllStringSubmatch(body, -1) {
		if n >= limit {
			break
		}
		link := strings.TrimSpace(m[1])
		title := html.UnescapeString(strings.TrimSpace(m[2]))
		if seen[link] || title == "" {
			continue
		}
		seen[link] = true

		desc := ""
		if d, _ := peopleFetchArticle(c, link); d != "" {
			desc = d
		}
		item := routeutils.NewItem(title, link, desc, peopleParseDate(m[3]))
		if item == nil {
			continue
		}
		routeutils.AddItem(feed, item)
		n++
	}
	return feed, nil
}

// peopleFetchArticle extracts article body text/HTML from an article page.
func peopleFetchArticle(c *ctxpkg.Context, link string) (string, string) {
	body, err := peopleProfile().Referer(link).Fetch(link).GetString(c.Parent(), c.Client())
	if err != nil {
		return "", ""
	}
	for _, marker := range []string{`id="rwb_zw"`, `class="rm_txt_con`, `id="rm_txt_zw"`, `class="show_text"`} {
		idx := strings.Index(body, marker)
		if idx < 0 {
			continue
		}
		start := strings.Index(body[idx:], ">")
		if start < 0 {
			continue
		}
		start += idx + 1
		end := strings.Index(body[start:], "</div>")
		if end < 0 {
			continue
		}
		content := strings.TrimSpace(body[start : start+end])
		if len(content) > 40 {
			return content, firstMatch(peopleDateRe, body)
		}
	}
	return "", firstMatch(peopleDateRe, body)
}

func firstMatch(re *regexp.Regexp, s string) string {
	if m := re.FindString(s); m != "" {
		return m
	}
	return ""
}

// peopleParseDate parses "2026年08月24日14:25" Beijing time.
func peopleParseDate(s string) time.Time {
	m := peopleDateRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}
	}
	y, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	d, _ := strconv.Atoi(m[3])
	h, _ := strconv.Atoi(m[4])
	mi, _ := strconv.Atoi(m[5])
	return time.Date(y, time.Month(mo), d, h, mi, 0, 0, time.FixedZone("CST", 8*60*60))
}
