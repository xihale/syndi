package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// doubanBookSubcats mirrors the 新书速递 sub-categories.
var doubanBookSubcats = map[string]string{
	"all":          "全部",
	"prose_poetry": "文学",
	"fiction":      "小说",
	"history":      "历史文化",
	"biography":    "社会纪实",
	"science":      "科学新知",
	"art":          "艺术设计",
	"business":     "商业经管",
	"comics":       "绘本漫画",
}

const doubanBookLatestURL = "https://book.douban.com/latest"

var doubanBookLatestRoute = routeutils.RouteSpec{
	Path:        "book/latest",
	Name:        "New Books",
	Example:     "douban/book/latest",
	Maintainers: []string{"xihale"},
	Description: "Douban new book express, all categories (豆瓣新书速递)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    6 * time.Hour,
	Handler:     DoubanBookLatestHandler,
}

var doubanBookLatestTypeRoute = routeutils.RouteSpec{
	Path:        "book/latest/:type",
	Name:        "New Books by Category",
	Example:     "douban/book/latest/fiction",
	Maintainers: []string{"xihale"},
	Description: "Douban new book express (豆瓣新书速递)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("type", "Sub-category: all (全部), prose_poetry (文学), fiction (小说), history (历史文化), biography (社会纪实), science (科学新知), art (艺术设计), business (商业经管), comics (绘本漫画)"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  DoubanBookLatestHandler,
}

// DoubanBookLatestHandler handles /douban/book/latest/:type?
func DoubanBookLatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	typ := routeutils.ParseEnum(c.Param("type"), "all", doubanCollectionKeys(doubanBookSubcats)...)
	ctx := c.Parent()

	apiURL := fmt.Sprintf("%s/subject_collection/new_book_%s/items?start=0&count=10&mode=collection&for_mobile=1", doubanRexxarAPI, typ)
	var resp doubanCollectionResp
	if err := doubanFetchJSON(ctx, c.Client(), apiURL, doubanBookLatestURL, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		doubanBookFeedTitle(typ),
		doubanBookLatestURL,
		"豆瓣读书新书速递",
	)
	doubanAppendBookItems(feed, resp.items())
	return feed, nil
}

func doubanBookFeedTitle(typ string) string {
	if name := doubanBookSubcats[typ]; typ != "all" && name != "" {
		return fmt.Sprintf("豆瓣新书速递-%s", name)
	}
	return "豆瓣新书速递"
}

func doubanAppendBookItems(feed *models.Feed, items []doubanCollectionItem) {
	for _, entry := range items {
		if item := buildDoubanBookItem(entry); item != nil {
			routeutils.AddItem(feed, item)
		}
	}
}

func buildDoubanBookItem(entry doubanCollectionItem) *models.Item {
	title := routeutils.CollapseWhitespace(entry.Title)
	link := doubanLink(&entry, "https://book.douban.com/subject/")
	if title == "" || link == "" {
		return nil
	}

	var sb strings.Builder
	if poster := entry.Poster(); poster != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s"/><br>`, html.EscapeString(poster)))
	}
	sb.WriteString(html.EscapeString(title) + "<br><br>")
	if info := routeutils.CollapseWhitespace(entry.CardSubtitle); info != "" {
		sb.WriteString(html.EscapeString(info) + "<br><br>")
	}
	if len(entry.Cards) > 0 {
		sb.WriteString(html.EscapeString(strings.TrimSpace(entry.Cards[0].Content)) + "<br><br>")
	}
	sb.WriteString(html.EscapeString(entry.RatingText()))

	item := routeutils.NewItem(title, link, sb.String(), time.Time{})
	if item == nil {
		return nil
	}
	item.GUID = "douban-book-" + firstNonEmpty(entry.ID, doubanIDFromLink(link))
	return item
}
