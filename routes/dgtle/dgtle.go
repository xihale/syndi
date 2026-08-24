package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"

	"strings"
	"sync"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const dgBaseURL = "https://www.dgtle.com"

// dgProfile disguises requests against dgtle.com.
var dgProfile = disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer(dgBaseURL + "/")

// dgImgSizeSuffixRe strips lazy-load size suffixes like _1800_500_w.
var dgImgSizeSuffixRe = regexp.MustCompile(`_\d+_\d+_w`)

// --- Upstream payload types ---

type dgUser struct {
	Username   dgFlexString `json:"username"`
	UserID     dgFlexString `json:"user_id"`
	AvatarPath dgFlexString `json:"avatar_path"`
}

type dgTagInfo struct {
	Title dgFlexString `json:"title"`
}

type dgNewsItem struct {
	ID           dgFlexString `json:"id"`
	Title        dgFlexString `json:"title"`
	Content      dgFlexString `json:"content"`
	Cover        dgFlexString `json:"cover"`
	CreatedAt    dgFlexInt64  `json:"created_at"`
	LiveStatus   *dgFlexInt64 `json:"live_status"`
	Column       dgFlexString `json:"column"`
	CategoryName dgFlexString `json:"category_name"`
	User         *dgUser      `json:"user"`
	UserID       dgFlexString `json:"user_id"`
	UserName     dgFlexString `json:"user_name"`
}

type dgFeedItem struct {
	ID         dgFlexString      `json:"id"`
	Content    dgFlexString      `json:"content"`
	ImgsURL    dgFlexStringSlice `json:"imgs_url"`
	URL        dgFlexString      `json:"url"`
	CreatedAt  dgFlexInt64       `json:"created_at"`
	UpdatedAt  dgFlexInt64       `json:"updated_at"`
	EncodeUID  dgFlexString      `json:"encode_uid"`
	UserID     dgFlexString      `json:"user_id"`
	UserName   dgFlexString      `json:"user_name"`
	AvatarPath dgFlexString      `json:"avatar_path"`
	TagsInfo   []dgTagInfo       `json:"tags_info"`
	LiveStatus *dgFlexInt64      `json:"live_status"`
}

type dgListResp struct {
	Status string `json:"status"`
	Data   struct {
		DataList json.RawMessage `json:"dataList"`
	} `json:"data"`
}

func dgParseList[T any](raw json.RawMessage) []T {
	var list []T
	if len(raw) > 0 && json.Unmarshal(raw, &list) == nil {
		return list
	}
	return nil
}

// --- Route specs ---

var dgtleNewsRoute = routeutils.RouteSpec{
	Path:        "news",
	Name:        "Dgtle News",
	Example:     "dgtle/news",
	Maintainers: []string{"xihale"},
	Description: "Latest entries from Dgtle Whale News (数字尾巴鲸闻)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "item count, default 20, max 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  DgtleNewsHandler,
}

var dgtleNewsCategoryRoute = routeutils.RouteSpec{
	Path:        "news/:id",
	Name:        "Dgtle News Category",
	Example:     "dgtle/news/396",
	Maintainers: []string{"xihale"},
	Description: "Dgtle Whale News by category. IDs: latest=0, live=395, news=396, daily quote=388",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "category ID, see description"),
		routeutils.OptionalParam("limit", "item count, default 20, max 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  DgtleNewsHandler,
}

var dgtleFeedRoute = routeutils.RouteSpec{
	Path:        "feed",
	Name:        "Dgtle Feed",
	Example:     "dgtle/feed",
	Maintainers: []string{"xihale"},
	Description: "Hot community dynamics from Dgtle Feed (数字尾巴兴趣动态)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "item count, default 20, max 50"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  DgtleFeedHandler,
}

// Routes lists all dgtle route specs in this package.
var Routes = []routeutils.RouteSpec{
	dgtleNewsRoute,
	dgtleNewsCategoryRoute,
	dgtleFeedRoute,
}

// --- Handlers ---

// DgtleNewsHandler handles /dgtle/news and /dgtle/news/:id (same handler).
func DgtleNewsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	if id == "" {
		id = "0"
	}
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()

	apiURL := fmt.Sprintf("%s/news/getNewsIndexList/%s", dgBaseURL, id)
	data, err := routeutils.FetchBytesWithHeaders(ctx, c.Client(), apiURL, dgProfile.Headers(apiURL))
	if err != nil {
		return nil, err
	}
	var resp dgListResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("dgtle: invalid news JSON: %w", err)
	}
	list := dgParseList[dgNewsItem](resp.Data.DataList)
	if len(list) > limit {
		list = list[:limit]
	}

	feedTitle := "数字尾巴 - 鲸闻"
	switch id {
	case "395":
		feedTitle = "数字尾巴 - 鲸闻直播"
	case "396":
		feedTitle = "数字尾巴 - 鲸闻资讯"
	case "388":
		feedTitle = "数字尾巴 - 每日一言"
	}
	feed := routeutils.NewFeed(feedTitle, dgBaseURL+"/news", "数字尾巴鲸闻最新内容")

	items := make([]*models.Item, len(list))
	var sem = make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, ni := range list {
		wg.Add(1)
		go func(idx int, ni dgNewsItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			items[idx] = dgBuildNewsItem(c, ni)
		}(i, ni)
	}
	wg.Wait()
	for _, item := range items {
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

// dgBuildNewsItem maps one whale-news entry to an item and fetches the detail
// page for the full article content.
func dgBuildNewsItem(c *ctxpkg.Context, ni dgNewsItem) *models.Item {
	id := ni.ID.String()
	if id == "" {
		return nil
	}
	title := strings.TrimSpace(stripDGTags(ni.Title.String()))
	contentPlain := stripDGTags(ni.Content.String())
	if title == "" {
		title = truncateRunes(contentPlain, 80)
	}
	if title == "" {
		return nil
	}

	// Verified against upstream: news-<id>-1.html serves the detail page for
	// whale-news entries (资讯 and 每日一言 alike); live entries use live-.
	prefix := "news"
	if ni.LiveStatus != nil {
		prefix = "live"
	}
	link := fmt.Sprintf("%s/%s-%s-1.html", dgBaseURL, prefix, id)

	desc := dgFigureHTML(ni.Cover.String())
	if contentPlain != "" {
		desc += "<p>" + html.EscapeString(contentPlain) + "</p>"
	}

	item := &models.Item{
		Title:       title,
		Link:        link,
		Description: desc,
		PubDate:     time.Unix(ni.CreatedAt.Int(), 0),
		GUID:        "dgtle-" + id,
	}
	authorName, authorUID := "", ""
	if ni.User != nil {
		authorName = ni.User.Username.String()
		authorUID = ni.User.UserID.String()
	}
	if authorName == "" {
		authorName = ni.UserName.String()
	}
	if authorUID == "" {
		authorUID = ni.UserID.String()
	}
	if authorName != "" {
		routeutils.SetItemAuthor(item, authorName, "", dgBaseURL+"/user?uid="+authorUID)
	}

	page, err := dgProfile.Fetch(link).GetString(c.Parent(), c.Client())
	if err != nil {
		return item
	}
	detail, err := parser.LoadString(page)
	if err != nil {
		return item
	}
	content := dgDetailContent(detail)
	if content != "" {
		item.Description = content
	}
	return item
}

// dgDetailContent extracts cleaned article HTML from a Dgtle detail page.
func dgDetailContent(doc *parser.Document) string {
	doc.Each("figure", func(_ int, fig *parser.Selection) {
		img := fig.Find("img").First()
		src := img.AttrOr("data-original", "")
		if src == "" {
			src = img.AttrOr("src", "")
		}
		src = dgImgSizeSuffixRe.ReplaceAllString(src, "")
		if src != "" {
			fig.ReplaceWithHtml(`<img src="` + html.EscapeString(src) + `" alt=""/>`)
			return
		}
		fig.Remove()
	})
	doc.Find("div.logo, p.tip, p.dgtle").Remove()

	for _, sel := range []string{
		"div.whale_news_detail-daily-content",
		"div#articleContent",
		"div.forum-viewthread-article-box",
	} {
		if s := doc.FindSelector(sel); s != nil && s.Length() > 0 {
			if h, err := s.Html(); err == nil && strings.TrimSpace(h) != "" {
				return h
			}
		}
	}
	return ""
}

// DgtleFeedHandler handles /dgtle/feed (兴趣 hot dynamics).
func DgtleFeedHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()

	apiURL := dgBaseURL + "/feed/getHotDynamic?last_id=0"
	data, err := routeutils.FetchBytesWithHeaders(ctx, c.Client(), apiURL, dgProfile.Headers(apiURL))
	if err != nil {
		return nil, err
	}
	var resp dgListResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("dgtle: invalid feed JSON: %w", err)
	}
	list := dgParseList[dgFeedItem](resp.Data.DataList)

	feed := routeutils.NewFeed("数字尾巴 - 兴趣", dgBaseURL+"/feed", "数字尾巴兴趣社区热门动态")
	n := 0
	for _, fi := range list {
		if n >= limit {
			break
		}
		id := fi.ID.String()
		title := truncateRunes(strings.TrimSpace(stripDGTags(fi.Content.String())), 80)
		if title == "" || id == "" {
			continue
		}
		link := fi.URL.String()
		if link == "" {
			link = "/inst-" + id + "-1.html"
		}
		if !strings.HasPrefix(link, "http") {
			link = dgBaseURL + link
		}

		var b strings.Builder
		for _, img := range fi.ImgsURL {
			b.WriteString(dgFigure(img))
		}
		if text := strings.TrimSpace(fi.Content.String()); text != "" {
			b.WriteString("<p>" + strings.ReplaceAll(html.EscapeString(text), "\n", "<br/>") + "</p>")
		}

		pubDate := time.Unix(fi.CreatedAt.Int(), 0)
		item := routeutils.NewItem(title, link, b.String(), pubDate)
		if item == nil {
			continue
		}
		item.GUID = "dgtle-" + id
		if u := fi.UpdatedAt.Int(); u > 0 && u != fi.CreatedAt.Int() {
			routeutils.SetUpdated(item, time.Unix(u, 0))
		}
		uid := fi.EncodeUID.String()
		if uid == "" {
			uid = fi.UserID.String()
		}
		name := fi.UserName.String()
		if name != "" {
			routeutils.SetItemAuthor(item, name, "", dgBaseURL+"/user?uid="+uid)
		}
		var cats []string
		for _, t := range fi.TagsInfo {
			if v := t.Title.String(); v != "" {
				cats = append(cats, v)
			}
		}
		routeutils.SetCategories(item, cats...)
		routeutils.AddItem(feed, item)
		n++
	}
	return feed, nil
}

// --- Shared helpers ---

var (
	dgTagRe   = regexp.MustCompile(`<[^>]*>`)
	dgSpaceRe = regexp.MustCompile(`\s+`)
)

// stripDGTags removes markdown/HTML tags and collapses whitespace.
func stripDGTags(s string) string {
	s = dgTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return dgSpaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
}

// truncateRunes cuts s to at most max runes, appending an ellipsis.
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// dgFigure renders one image as an HTML figure fragment; "" when src is empty.
func dgFigure(src string) string {
	if src == "" {
		return ""
	}
	return `<figure><img src="` + html.EscapeString(src) + `" alt=""/></figure>`
}

// dgFigureHTML is kept for readability at call sites.
func dgFigureHTML(src string) string { return dgFigure(src) }
