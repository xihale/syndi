package routes

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// ---------------- 收藏夹 /zhihu/collection/:id ----------------

type zhihuCollectionItem struct {
	ContentType string                `json:"type"` // answer | article | zvideo | pin
	Content     zhihuMomentTargetData `json:"content"`
}

type zhihuCollectionItemsResp struct {
	Paging struct {
		IsEnd  bool `json:"is_end"`
		Totals int  `json:"totals"`
	} `json:"paging"`
	Data []zhihuCollectionItem `json:"data"`
}

const (
	zhihuCollectionPageLimit = 20
	zhihuCollectionMaxItems  = 500
)

var zhihuCollectionRoute = routeutils.RouteSpec{
	Path:        "collection/:id",
	Name:        "知乎收藏夹",
	Example:     "zhihu/collection/26444956",
	Maintainers: []string{"xihale"},
	Description: "公开收藏夹的内容",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true, EnvDeps: []string{zhihuCookiesEnv}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "收藏夹 id，可在收藏夹页面 URL 中找到"),
		routeutils.OptionalParam("limit", "条数，默认 20，上限 500"),
		routeutils.OptionalParam("all", "获取全部收藏内容，任意真值为打开"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  ZhihuCollectionHandler,
}

// ZhihuCollectionHandler handles /zhihu/collection/:id
func ZhihuCollectionHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	fetchAll := routeutils.ParseBool(c.QueryParam("all"), false)
	limitCap := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, zhihuCollectionMaxItems)
	ctx := c.Parent()
	if err := requireZhihuCookies(); err != nil {
		return nil, err
	}
	if fetchAll {
		limitCap = zhihuCollectionMaxItems
	}
	pageURL := fmt.Sprintf("https://www.zhihu.com/collection/%s", id)

	title, description := fetchZhihuCollectionMeta(ctx, c, id)

	feed := routeutils.NewFeed(title, pageURL, description)
	offset := 0
	for len(feed.Items) < limitCap {
		q := url.Values{}
		q.Set("offset", strconv.Itoa(offset))
		q.Set("limit", strconv.Itoa(zhihuCollectionPageLimit))
		apiURL := fmt.Sprintf("%s/collections/%s/items?%s", zhihuAPIBase, id, q.Encode())
		var resp zhihuCollectionItemsResp
		if err := zhihuProfile(pageURL).Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
			if offset == 0 {
				return nil, err
			}
			break // 后续页失败时保留已取到的内容
		}
		for _, entry := range resp.Data {
			item := mapZhihuCollectionEntry(entry)
			if item != nil && len(feed.Items) < limitCap {
				routeutils.AddItem(feed, item)
			}
		}
		if resp.Paging.IsEnd || len(resp.Data) == 0 {
			break
		}
		offset += zhihuCollectionPageLimit
	}
	return feed, nil
}

// mapZhihuCollectionEntry converts one favorited content to a feed item.
func mapZhihuCollectionEntry(entry zhihuCollectionItem) *models.Item {
	t := entry.Content
	switch t.Type {
	case "answer":
		link := t.URL
		if link == "" {
			link = fmt.Sprintf("https://www.zhihu.com/question/%d/answer/%d", t.Question.ID, t.ID)
		}
		desc := processZhihuContent(t.Content.String())
		item := routeutils.NewItem(t.Question.Title, link, desc, time.Unix(firstNonZero(t.UpdatedTime, t.CreatedTime), 0))
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-answer-%d", t.ID)
			routeutils.SetItemAuthor(item, t.Author.Name, "", "")
		}
		return item
	case "article", "zvideo":
		link := t.URL
		if link == "" {
			link = fmt.Sprintf("https://zhuanlan.zhihu.com/p/%d", t.ID)
		}
		desc := processZhihuContent(t.Content.String())
		if t.Type == "zvideo" && strings.TrimSpace(desc) == "" {
			desc = "<p>视频内容请跳转至原页面观看</p>"
		}
		item := routeutils.NewItem(t.Title, link, desc, time.Unix(firstNonZero(t.UpdatedTime, t.Created), 0))
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-%s-%d", t.Type, t.ID)
			routeutils.SetItemAuthor(item, t.Author.Name, "", "")
		}
		return item
	case "pin":
		link := fmt.Sprintf("https://www.zhihu.com/pin/%d", t.ID)
		var parts []string
		for _, pc := range t.Content.Parts {
			if pc.Type == "text" && pc.OwnText != "" {
				parts = append(parts, "<p>"+html.EscapeString(pc.OwnText)+"</p>")
			}
		}
		if len(parts) == 0 && t.ExcerptTitle != "" {
			parts = append(parts, "<p>"+html.EscapeString(t.ExcerptTitle)+"</p>")
		}
		item := routeutils.NewItem(firstNonEmpty(t.ExcerptTitle, truncatePlain(strings.Join(plainTexts(parts), " "), 60)),
			link, strings.Join(parts, ""), time.Unix(firstNonZero(t.UpdatedTime, t.CreatedTime), 0))
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-pin-%d", t.ID)
			routeutils.SetItemAuthor(item, t.Author.Name, "", "")
		}
		return item
	default:
		return nil
	}
}

// fetchZhihuCollectionMeta scrapes the public collection page for title and
// description; falls back to generic wording if the page markup changes.
func fetchZhihuCollectionMeta(ctx context.Context, c *ctxpkg.Context, id string) (title, description string) {
	generic := "知乎收藏夹 " + id
	pageURL := fmt.Sprintf("https://www.zhihu.com/collection/%s", id)
	doc, err := zhihuWebProfile(pageURL).Fetch(pageURL).GetHTML(ctx, c.Client())
	if err != nil || doc == nil {
		return generic + " - 知乎收藏夹", ""
	}
	t := strings.TrimSpace(doc.Text(".CollectionDetailPageHeader-title"))
	d := strings.TrimSpace(doc.Text(".CollectionDetailPageHeader-description"))
	if t == "" {
		t = generic
	} else {
		t += " - 知乎收藏夹"
	}
	return t, d
}

func plainTexts(htmlParts []string) []string {
	out := make([]string, 0, len(htmlParts))
	for _, p := range htmlParts {
		out = append(out, html.UnescapeString(strings.NewReplacer("<p>", "", "</p>", "").Replace(p)))
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncatePlain(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}

// ---------------- 知乎周刊 /zhihu/weekly ----------------

var zhihuWeeklyRoute = routeutils.RouteSpec{
	Path:        "weekly",
	Name:        "知乎周刊",
	Example:     "zhihu/weekly",
	Maintainers: []string{"xihale"},
	Description: "知乎书店免费周刊列表",
	Categories:  []models.Category{{Name: "study"}},
	CacheTTL:    24 * time.Hour,
	Handler:     ZhihuWeeklyHandler,
}

// ZhihuWeeklyHandler handles /zhihu/weekly
func ZhihuWeeklyHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()

	const link = "https://www.zhihu.com/pub/weekly"
	doc, err := zhihuWebProfile(link).Fetch(link).GetHTML(ctx, c.Client())
	if err != nil || doc == nil {
		return nil, fmt.Errorf("获取周刊页面失败: %w", err)
	}

	feed := routeutils.NewFeed("知乎周刊", link, doc.Text("p.Weekly-description"))
	doc.Each("div.PubBookListItem", func(_ int, s *parser.Selection) {
		title := strings.TrimSpace(s.Find("span.PubBookListItem-title").TextTrim())
		href, ok := s.Find(`a[class*="PubBookListItem-button"]`).Attr("href")
		if title == "" || !ok || href == "" {
			return
		}
		itemLink := resolveZhihuLink(href)
		desc := strings.TrimSpace(s.Find("div.PubBookListItem-description").TextTrim())
		author := strings.TrimSpace(s.Find("span.PubBookListItem-author").TextTrim())

		var sb strings.Builder
		if author != "" {
			sb.WriteString("<p>" + html.EscapeString(author) + "</p>")
		}
		if desc != "" {
			sb.WriteString("<p>" + html.EscapeString(desc) + "</p>")
		}
		item := routeutils.NewItem(title, itemLink, sb.String(), time.Time{})
		if item != nil {
			item.GUID = itemLink
			routeutils.SetItemAuthor(item, author, "", "")
			routeutils.AddItem(feed, item)
		}
	})
	return feed, nil
}

func resolveZhihuLink(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	return "https://www.zhihu.com" + href
}
