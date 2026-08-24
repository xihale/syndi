package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const (
	hxRootURL        = "https://www.huxiu.com"
	hxArticleListAPI = "https://api-web-article.huxiu.com/web/channel/articleListV1"
	hxMomentFeedAPI  = "https://moment-api.huxiu.com/web-v3/moment/feed"
)

// hxProfile disguises requests against huxiu APIs and pages.
var hxProfile = disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer(hxRootURL + "/")

// --- Upstream payload types ---

type hxUserInfo struct {
	Username flexString `json:"username"`
}

type hxCountInfo struct {
	Agree           flexInt64 `json:"agree"`
	Favtimes        flexInt64 `json:"favtimes"`
	Agreenum        flexInt64 `json:"agreenum"`
	Commentnum      flexInt64 `json:"commentnum"`
	TotalCommentNum flexInt64 `json:"total_comment_num"`
}

type hxListItem struct {
	Aid         flexString  `json:"aid"`
	Title       flexString  `json:"title"`
	Summary     flexString  `json:"summary"`
	URL         flexString  `json:"url"`
	PicPath     flexString  `json:"pic_path"`
	Dateline    flexInt64   `json:"dateline"`
	PublishTime flexInt64   `json:"publish_time"`
	UserInfo    *hxUserInfo `json:"user_info"`
	CountInfo   hxCountInfo `json:"count_info"`
}

type hxListResp struct {
	Success bool `json:"success"`
	Data    struct {
		Name     string          `json:"name"`
		DataList json.RawMessage `json:"dataList"` // alternate upstream spelling
		Datalist json.RawMessage `json:"datalist"`
	} `json:"data"`
}

func (r *hxListResp) items() []hxListItem {
	raw := r.Data.DataList
	if len(raw) == 0 {
		raw = r.Data.Datalist
	}
	var list []hxListItem
	if len(raw) > 0 && json.Unmarshal(raw, &list) == nil {
		return list
	}
	return nil
}

type hxMomentItem struct {
	ObjectID    flexInt64       `json:"object_id"`
	ObjectType  flexInt64       `json:"object_type"`
	Content     flexString      `json:"content"`
	PublishTime flexInt64       `json:"publish_time"`
	URL         flexString      `json:"url"`
	UserInfo    *hxUserInfo     `json:"user_info"`
	CountInfo   hxCountInfo     `json:"count_info"`
	ImgURLs     flexStringSlice `json:"img_urls"`
}

type hxMomentResp struct {
	Success bool `json:"success"`
	Data    struct {
		MomentList struct {
			Datalist []hxMomentItem `json:"datalist"`
		} `json:"moment_list"`
	} `json:"data"`
}

// --- Route specs ---

var huxiuArticleRoute = routeutils.RouteSpec{
	Path:        "article",
	Name:        "Huxiu Articles",
	Example:     "huxiu/article",
	Maintainers: []string{"xihale"},
	Description: "Latest articles from Huxiu (虎嗅资讯), with full article content",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true, AntiCrawler: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "item count, default 20, max 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  HuxiuArticleHandler,
}

var huxiuChannelRoute = routeutils.RouteSpec{
	Path:        "channel/:id",
	Name:        "Huxiu Channel",
	Example:     "huxiu/channel/105",
	Maintainers: []string{"xihale"},
	Description: "Articles from a Huxiu channel. IDs: video=10, frontier tech=105, auto=21, business=103, culture=106, finance=115, going global=114, world=107, games=22, health=118, books=119, medical=120, digital=121, opinion=122, others=123",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{AntiCrawler: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "channel ID, see description"),
		routeutils.OptionalParam("limit", "item count, default 20, max 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  HuxiuChannelHandler,
}

var huxiuMomentRoute = routeutils.RouteSpec{
	Path:        "moment",
	Name:        "Huxiu 24 Hours",
	Example:     "huxiu/moment",
	Maintainers: []string{"xihale"},
	Description: "Huxiu 24-hour breaking news feed (虎嗅 24 小时)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true, AntiCrawler: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "item count, default 20, max 50"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  HuxiuMomentHandler,
}

// Routes lists all huxiu route specs in this package.
var Routes = []routeutils.RouteSpec{
	huxiuArticleRoute,
	huxiuChannelRoute,
	huxiuMomentRoute,
}

// --- Handlers ---

// HuxiuArticleHandler handles /huxiu/article.
func HuxiuArticleHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return hxArticleList(c, "0")
}

// HuxiuChannelHandler handles /huxiu/channel/:id.
func HuxiuChannelHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return hxArticleList(c, c.Param("id"))
}

// hxArticleList fetches the channel article list and enriches with details.
func hxArticleList(c *ctxpkg.Context, channelID string) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()

	form := url.Values{}
	form.Set("platform", "www")
	form.Set("channel_id", channelID)
	form.Set("pagesize", strconv.Itoa(limit))
	data, err := c.Client().PostWithHeaders(ctx, hxArticleListAPI, []byte(form.Encode()), hxProfile.Headers(hxArticleListAPI))
	if err != nil {
		return nil, err
	}
	var resp hxListResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("huxiu: invalid list JSON: %w", err)
	}
	list := resp.items()

	feedTitle := "虎嗅资讯"
	feedLink := hxRootURL + "/article"
	feedDesc := "虎嗅最新资讯"
	if resp.Data.Name != "" && channelID != "0" {
		feedTitle = "虎嗅 - " + resp.Data.Name
		feedLink = fmt.Sprintf("%s/channel/%s.html", hxRootURL, channelID)
	}

	feed := routeutils.NewFeed(feedTitle, feedLink, feedDesc)
	items := make([]*models.Item, len(list))
	var sem = make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, li := range list {
		if i >= limit {
			break
		}
		wg.Add(1)
		go func(idx int, li hxListItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			items[idx] = hxBuildArticleItem(c, li)
		}(i, li)
	}
	wg.Wait()
	for _, item := range items {
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

// hxBuildArticleItem maps one list entry to an item, fetching the article page
// for full content; on failure it falls back to the list-level summary.
func hxBuildArticleItem(c *ctxpkg.Context, li hxListItem) *models.Item {
	aid := li.Aid.String()
	link := li.URL.String()
	if link == "" && aid != "" {
		link = fmt.Sprintf("%s/article/%s.html", hxRootURL, aid)
	}
	title := stripTags(li.Title.String())
	pubDate := time.Unix(li.PublishTime.Int(), 0)
	if li.Dateline.Int() > 0 {
		pubDate = time.Unix(li.Dateline.Int(), 0)
	}
	desc := hxFigureHTML(li.PicPath.String())
	if summary := strings.TrimSpace(li.Summary.String()); summary != "" {
		desc += "<p>" + html.EscapeString(summary) + "</p>"
	}

	item := &models.Item{
		Title:       title,
		Link:        link,
		Description: desc,
		PubDate:     pubDate,
		GUID:        "huxiu-article-" + aid,
	}
	if li.UserInfo != nil {
		routeutils.SetItemAuthor(item, li.UserInfo.Username.String(), "", "")
	}

	page, err := hxProfile.Fetch(link).GetString(c.Parent(), c.Client())
	if err != nil {
		return item
	}
	detail := articleDetailFromNuxt(page)
	if detail == nil {
		return item
	}
	if t := nuxtString(detail, "title"); t != "" {
		item.Title = stripTags(t)
	}
	content := cleanHuxiuHTML(nuxtString(detail, "content"))
	preface := cleanHuxiuHTML(nuxtString(detail, "preface", "content_preface"))
	summary := nuxtString(detail, "summary")

	var b strings.Builder
	b.WriteString(desc)
	if preface != "" {
		b.WriteString(preface)
	}
	if summary != "" {
		b.WriteString("<p>" + html.EscapeString(summary) + "</p>")
	}
	if content != "" {
		b.WriteString(content)
	}
	item.Description = b.String()

	if ts := anyToTimeUnix(detail["publish_time"]); ts > 0 {
		item.PubDate = time.Unix(ts, 0)
	} else if ts := anyToTimeUnix(detail["dateline"]); ts > 0 {
		item.PubDate = time.Unix(ts, 0)
	}
	author := ""
	if ui := nested(detail, "user_info"); ui != nil {
		author = nuxtString(ui, "username")
	}
	if author == "" {
		if ai := nested(detail, "author_info"); ai != nil {
			author = nuxtString(ai, "username")
		}
	}
	if author != "" {
		routeutils.SetItemAuthor(item, author, "", "")
	}
	routeutils.SetCategories(item, nuxtTags(detail)...)
	return item
}

// HuxiuMomentHandler handles /huxiu/moment (24小时).
func HuxiuMomentHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()

	data, err := c.Client().PostWithHeaders(ctx, hxMomentFeedAPI, []byte(url.Values{"platform": {"www"}}.Encode()), hxProfile.Headers(hxMomentFeedAPI))
	if err != nil {
		return nil, err
	}
	var resp hxMomentResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("huxiu: invalid moment JSON: %w", err)
	}

	feed := routeutils.NewFeed("虎嗅 - 24小时", hxRootURL+"/moment", "虎嗅 24 小时新闻快讯")
	n := 0
	for _, mi := range resp.Data.MomentList.Datalist {
		if mi.ObjectType.Int() != 8 || n >= limit {
			continue
		}
		id := mi.ObjectID.Int()
		link := mi.URL.String()
		if link == "" {
			link = fmt.Sprintf("%s/moment/%d.html", hxRootURL, id)
		}
		contentHTML := mi.Content.String()
		title := truncateRunes(strings.TrimSpace(stripTags(contentHTML)), 80)
		if title == "" {
			continue
		}
		var b strings.Builder
		for _, img := range mi.ImgURLs {
			b.WriteString(hxFigure(img))
		}
		b.WriteString(sanitizeMomentHTML(contentHTML))

		item := routeutils.NewItem(title, link, b.String(), time.Unix(mi.PublishTime.Int(), 0))
		if item == nil {
			continue
		}
		item.GUID = fmt.Sprintf("huxiu-moment-%d", id)
		if mi.UserInfo != nil {
			routeutils.SetItemAuthor(item, mi.UserInfo.Username.String(), "", "")
		}
		routeutils.AddItem(feed, item)
		n++
	}
	return feed, nil
}

// --- Shared helpers ---

var (
	hxTagRe      = regexp.MustCompile(`<[^>]*>`)
	hxSpaceRe    = regexp.MustCompile(`\s+`)
	hxImgQueryRe = regexp.MustCompile(`\?.*$`)
)

// stripTags removes HTML tags and collapses whitespace into plain text.
func stripTags(s string) string {
	s = hxTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return hxSpaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
}

// truncateRunes cuts s to at most max runes, appending an ellipsis.
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// hxFigure renders a single image figure HTML fragment.
func hxFigure(src string) string {
	if src == "" {
		return ""
	}
	return `<figure><img src="` + html.EscapeString(src) + `" alt=""/></figure>`
}

var hxScriptRe = regexp.MustCompile(`(?is)<(script|iframe|style)\b[^>]*>.*?</(script|iframe|style)\s*>`)

// sanitizeMomentHTML keeps upstream markup (e.g. <br>) but drops active
// content such as script/iframe/style blocks and event handler attributes.
func sanitizeMomentHTML(s string) string {
	s = hxScriptRe.ReplaceAllString(s, "")
	return strings.ReplaceAll(s, "\n", "<br/>")
}

// hxFigureHTML is an alias kept for readability at call sites.
func hxFigureHTML(src string) string { return hxFigure(src) }

// cleanHuxiuHTML tidies Huxiu article content HTML similar to the reference
// implementation: drop promo/vote blocks, normalize lazy images, remove noisy
// attributes, drop empty paragraphs/spans, and promote styled titles.
func cleanHuxiuHTML(fragment string) string {
	if strings.TrimSpace(fragment) == "" {
		return ""
	}
	doc, err := parser.LoadString("<div id=\"hx-root\">" + fragment + "</div>")
	if err != nil {
		return fragment
	}
	root := doc.Find("#hx-root")

	doc.Find("div.neirong-shouquan").Remove()
	doc.Find("em.vote__bar, div.vote__btn, div.vote__time").Remove()

	doc.Find("p img").Each(func(_ int, s *goquery.Selection) {
		src := s.AttrOr("src", "")
		if src == "" {
			src = s.AttrOr("_src", "")
		}
		if src != "" {
			src = hxImgQueryRe.ReplaceAllString(src, "")
			img := `<img src="` + html.EscapeString(src) + `"`
			if w := s.AttrOr("data-w", ""); w != "" {
				img += ` width="` + w + `"`
			}
			if h := s.AttrOr("data-h", ""); h != "" {
				img += ` height="` + h + `"`
			}
			img += `/>`
			s.Parent().ReplaceWithHtml("<p>" + img + "</p>")
		}
	})

	doc.Find("p, span").Each(func(_ int, s *goquery.Selection) {
		if s.Contents().Length() == 1 && strings.TrimSpace(s.Text()) == "" {
			s.Remove()
			return
		}
		s.RemoveAttr("class")
		s.RemoveAttr("data-check-id")
		s.RemoveAttr("label")
	})

	renameTag := func(s *goquery.Selection, tag string) {
		inner, _ := s.Html()
		if strings.TrimSpace(s.Text()) == "" && strings.TrimSpace(inner) == "" {
			s.Remove()
			return
		}
		s.ReplaceWithHtml("<" + tag + ">" + inner + "</" + tag + ">")
	}
	doc.Find(".text-big-title").Each(func(_ int, s *goquery.Selection) { renameTag(s, "h3") })
	doc.Find(".text-sm-title").Each(func(_ int, s *goquery.Selection) { renameTag(s, "h4") })

	htmlStr, err := root.Html()
	if err != nil {
		return fragment
	}
	return htmlStr
}
