package routes

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const doubanExploreURL = "https://www.douban.com/explore/recommend"

var doubanExploreRoute = routeutils.RouteSpec{
	Path:        "explore",
	Name:        "Douban Explore",
	Example:     "douban/explore",
	Maintainers: []string{"xihale"},
	Description: "Douban explore recommended posts (浏览发现-为你推荐)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    30 * time.Minute,
	Handler:     DoubanExploreHandler,
}

// DoubanExploreHandler handles /douban/explore
//
// The main 人气创作 tab of www.douban.com/explore is now rendered client-side
// only; the 为你推荐 tab still ships server-rendered HTML and is used here.
func DoubanExploreHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()

	doc, err := doubanFetchHTML(ctx, c.Client(), doubanExploreURL, "https://www.douban.com/explore")
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"豆瓣-浏览发现",
		doubanExploreURL,
		"豆瓣浏览发现·为你推荐",
	)
	doubanAppendExploreItems(feed, doc)
	return feed, nil
}

func doubanAppendExploreItems(feed *models.Feed, doc *parser.Document) {
	doc.Each("li.item", func(_ int, sel *parser.Selection) {
		if item := parseDoubanExploreItem(sel); item != nil {
			routeutils.AddItem(feed, item)
		}
	})
}

// doubanBGImagePattern extracts url('...') from inline cover styles.
var doubanBGImagePattern = regexp.MustCompile(`\('(.*?)'\)`)

// parseDoubanExploreItem converts one recommended entry into an item.
func parseDoubanExploreItem(sel *parser.Selection) *models.Item {
	titleSel := sel.Find(".bd .content .title a")
	title := routeutils.CollapseWhitespace(titleSel.TextTrim())
	link := titleSel.AttrOr("href", "")

	descText := ""
	pSel := sel.Find(".bd .content p")
	if pSel.Length() > 0 {
		descText = strings.TrimSpace(pSel.Text())
		if link == "" {
			link = pSel.Find("a").First().AttrOr("href", "")
			if title == "" {
				title = routeutils.CollapseWhitespace(pSel.Find("a").First().TextTrim())
			}
		}
	}

	pic := ""
	cover := sel.Find("a.cover")
	if style, ok := cover.Attr("style"); ok {
		if m := doubanBGImagePattern.FindStringSubmatch(style); m != nil {
			pic = m[1]
		}
	}

	author := ""
	if authorSel := sel.Find(".hd .usr-pic a"); authorSel.Length() > 0 {
		author = routeutils.CollapseWhitespace(authorSel.Last().TextTrim())
	}

	if title == "" && descText == "" {
		return nil
	}
	if title == "" {
		title = routeutils.Truncate(routeutils.CollapseWhitespace(descText), 40, "…")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("作者：%s<br>", html.EscapeString(author)))
	sb.WriteString(fmt.Sprintf("描述：%s<br>", html.EscapeString(descText)))
	if pic != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s"/>`, html.EscapeString(pic)))
	}

	item := routeutils.NewItem(title, link, sb.String(), time.Time{})
	if item == nil {
		return nil
	}
	if author != "" {
		routeutils.SetAuthor(item, author)
	}
	item.GUID = "douban-explore-" + firstNonEmpty(doubanIDFromLink(link), title)
	return item
}
