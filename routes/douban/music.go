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

const doubanMusicLatestURL = "https://music.douban.com/latest"

// doubanMusicAreas maps the latest-music area param to rexxar collections.
var doubanMusicAreas = map[string]struct{ name, path string }{
	"chinese":     {"华语新碟榜", "chinese"},
	"western":     {"欧美新碟榜", "occident"},
	"japankorean": {"日韩新碟榜", "japan_korea"},
}

var doubanMusicLatestRoute = routeutils.RouteSpec{
	Path:        "music/latest",
	Name:        "Latest Added Music",
	Example:     "douban/music/latest",
	Maintainers: []string{"xihale"},
	Description: "Music recently added to douban (豆瓣最新增加的音乐)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    6 * time.Hour,
	Handler:     DoubanMusicLatestHandler,
}

var doubanMusicLatestAreaRoute = routeutils.RouteSpec{
	Path:        "music/latest/:area",
	Name:        "Latest Music by Area",
	Example:     "douban/music/latest/chinese",
	Maintainers: []string{"xihale"},
	Description: "New album rankings by area (豆瓣最新增加的音乐)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("area", "Area: chinese (华语), western (欧美), japankorean (日韩)"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  DoubanMusicLatestHandler,
}

// DoubanMusicLatestHandler handles /douban/music/latest/:area?
func DoubanMusicLatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	area := strings.ToLower(strings.TrimSpace(c.Param("area")))
	ctx := c.Parent()

	if area == "" {
		doc, err := doubanFetchHTML(ctx, c.Client(), doubanMusicLatestURL, doubanWWWBaseURL+"/")
		if err != nil {
			return nil, err
		}
		feed := routeutils.NewFeed("豆瓣最新增加的音乐", doubanMusicLatestURL, "豆瓣最新增加的音乐")
		doubanAppendMusicLatestHTML(feed, doc)
		return feed, nil
	}

	info, ok := doubanMusicAreas[area]
	if !ok {
		return nil, fmt.Errorf("douban music/latest: unknown area %q (supported: chinese, western, japankorean)", area)
	}
	apiURL := fmt.Sprintf("%s/subject_collection/music_%s/items?os=ios&callback=&start=0&count=20&loc_id=0", doubanRexxarAPI, info.path)
	referer := doubanMobileURL + "/music/"
	var resp doubanCollectionResp
	if err := doubanFetchJSON(ctx, c.Client(), apiURL, referer, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("豆瓣最新增加的音乐-%s", info.name),
		referer+"new"+area,
		fmt.Sprintf("豆瓣最新增加的音乐-%s", info.name),
	)
	doubanAppendMusicCollection(feed, resp.items())
	return feed, nil
}

// doubanAppendMusicLatestHTML parses music.douban.com/latest list items.
func doubanAppendMusicLatestHTML(feed *models.Feed, doc *parser.Document) {
	doc.Each("li.dlist", func(_ int, sel *parser.Selection) {
		if item := parseDoubanMusicLatestItem(sel); item != nil {
			routeutils.AddItem(feed, item)
		}
	})
}

func parseDoubanMusicLatestItem(sel *parser.Selection) *models.Item {
	aSel := sel.Find("a.pl2")
	title := routeutils.CollapseWhitespace(aSel.TextTrim())
	link := aSel.AttrOr("href", "")
	if title == "" || link == "" {
		return nil
	}
	info := routeutils.CollapseWhitespace(sel.Find("p.pl").TextTrim())

	var sb strings.Builder
	if poster := sel.Find(".fleft img").AttrOr("src", ""); poster != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s"/><br>`, html.EscapeString(poster)))
	}
	sb.WriteString(html.EscapeString(title))
	if info != "" {
		sb.WriteString("<br>" + html.EscapeString(info))
	}

	item := routeutils.NewItem(title, link, sb.String(), time.Time{})
	if item == nil {
		return nil
	}
	item.GUID = "douban-music-latest-" + firstNonEmpty(doubanIDFromLink(link), title)
	return item
}

// doubanAppendMusicCollection renders m.douban.com collection entries.
func doubanAppendMusicCollection(feed *models.Feed, items []doubanCollectionItem) {
	for _, entry := range items {
		if item := buildDoubanMusicItem(entry); item != nil {
			routeutils.AddItem(feed, item)
		}
	}
}

func buildDoubanMusicItem(entry doubanCollectionItem) *models.Item {
	title := doubanItemTitle(&entry)
	if title == "" || entry.ID == "" {
		return nil
	}
	link := fmt.Sprintf("https://music.douban.com/subject/%s/", entry.ID)

	pubDate := time.Time{}
	if len(entry.Pubdate) > 0 {
		pubDate = doubanParseDate(entry.Pubdate[0])
	}

	var sb strings.Builder
	if poster := entry.Poster(); poster != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s"/><br>`, html.EscapeString(poster)))
	}
	if comment := routeutils.CollapseWhitespace(firstNonEmpty(entry.RecommendComment, entry.Info)); comment != "" {
		sb.WriteString(html.EscapeString(comment) + "<br>")
	}
	sb.WriteString("<strong>评分:</strong> " + html.EscapeString(entry.RatingText()))

	item := routeutils.NewItem(title, link, sb.String(), pubDate)
	if item == nil {
		return nil
	}
	item.GUID = "douban-music-" + entry.ID
	return item
}
