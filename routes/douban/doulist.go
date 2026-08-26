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

var doubanDoulistRoute = routeutils.RouteSpec{
	Path:        "doulist/:id",
	Name:        "Douban Doulist",
	Example:     "douban/doulist/37716774",
	Maintainers: []string{"xihale"},
	Description: "Items of a douban doulist (豆瓣豆列)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Doulist id, e.g. 37716774"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  DoubanDoulistHandler,
}

// DoubanDoulistHandler handles /douban/doulist/:id
func DoubanDoulistHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := doubanSanitizeKey(c.Param("id"), "")
	if id == "" {
		return nil, fmt.Errorf("douban doulist: invalid id %q", c.Param("id"))
	}

	target := fmt.Sprintf("%s/doulist/%s/", doubanWWWBaseURL, id)
	ctx := c.Parent()
	doc, err := doubanFetchHTML(ctx, c.Client(), target, target)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		firstNonEmpty(routeutils.CollapseWhitespace(doc.First("#content h1").TextTrim()), "豆瓣豆列"),
		target,
		routeutils.CollapseWhitespace(doc.First("div.doulist-about").TextTrim()),
	)
	doubanAppendDoulistItems(feed, doc)
	return feed, nil
}

func doubanAppendDoulistItems(feed *models.Feed, doc *parser.Document) {
	doc.Each("div.doulist-item", func(_ int, sel *parser.Selection) {
		if item := parseDoubanDoulistItem(sel); item != nil {
			routeutils.AddItem(feed, item)
		}
	})
}

// doubanDoulistTimestamp reads the entry timestamp; old pages put it in a
// span[title] inside <time>, new ones as the time's own text.
func doubanDoulistTimestamp(sel *parser.Selection) string {
	timeEl := sel.Find("div.ft .actions time")
	if value, ok := timeEl.Attr("title"); ok && strings.TrimSpace(value) != "" {
		return value
	}
	if span := timeEl.Find("span"); span.Length() > 0 {
		if value, ok := span.Attr("title"); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return strings.TrimSpace(timeEl.TextTrim())
}

// parseDoubanDoulistItem renders a doulist entry; entries come from several
// douban sources (movie/book/music subjects, notes, status broadcasts), the
// div.source text selects the layout.
func parseDoubanDoulistItem(sel *parser.Selection) *models.Item {
	source := routeutils.CollapseWhitespace(sel.Find("div.source").TextTrim())

	title, link, desc := doubanDoulistEntry(sel, source)

	var sb strings.Builder
	sb.WriteString(desc)

	timestamp := time.Time{}
	rawDate := doubanDoulistTimestamp(sel)
	timestamp = doubanParseDate(rawDate)

	item := routeutils.NewItem(title, link, sb.String(), timestamp)
	if item == nil {
		return nil
	}
	id := firstNonEmpty(sel.AttrOr("id", ""), doubanIDFromLink(link))
	item.GUID = "douban-doulist-" + firstNonEmpty(id, title)
	return item
}

// doubanDoulistEntry extracts title/link/description for the known layouts.
func doubanDoulistEntry(sel *parser.Selection, source string) (title, link, desc string) {
	switch {
	case strings.Contains(source, "豆瓣广播"):
		aSel := sel.Find("p.status-content > a")
		title = routeutils.CollapseWhitespace(aSel.TextTrim())
		link = aSel.AttrOr("href", "")
		desc = html.EscapeString(routeutils.CollapseWhitespace(sel.Find("span.status-recommend-text").TextTrim()))
	case strings.Contains(source, "豆瓣电影"), strings.Contains(source, "豆瓣读书"),
		strings.Contains(source, "豆瓣音乐"), source == "来自：豆瓣":
		subject := sel.Find("div.bd.doulist-subject")
		aSel := subject.Find("div.title a")
		title = routeutils.CollapseWhitespace(aSel.TextTrim())
		link = aSel.AttrOr("href", "")
		var sb strings.Builder
		if img := sel.Find("div.post a img").AttrOr("src", ""); img != "" {
			sb.WriteString(fmt.Sprintf(`<img width="100" src="%s">`, html.EscapeString(img)))
		}
		sb.WriteString(html.EscapeString(routeutils.CollapseWhitespace(subject.Find("div.abstract").TextTrim())))
		if comment := routeutils.CollapseWhitespace(sel.Find("div.ft div.comment-item.content").TextTrim()); comment != "" {
			sb.WriteString("<blockquote>" + html.EscapeString(comment) + "</blockquote>")
		}
		desc = sb.String()
	default:
		note := sel.Find("div.bd.doulist-note")
		aSel := note.Find("div.title a")
		title = routeutils.CollapseWhitespace(aSel.TextTrim())
		link = aSel.AttrOr("href", "")
		desc = html.EscapeString(routeutils.CollapseWhitespace(note.Find("div.abstract").TextTrim()))
	}
	return title, link, desc
}
