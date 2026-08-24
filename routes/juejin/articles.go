package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// juejinProfile disguises requests against the juejin web APIs.
var juejinProfile = disguise.Chrome().JSONAccept().Lang("zh-CN,zh;q=0.9")

// --- Upstream payload types (api.juejin.cn) ---

type juejinResp struct {
	ErrNo  int             `json:"err_no"`
	ErrMsg string          `json:"err_msg"`
	Data   json.RawMessage `json:"data"`
}

type juejinCategoryBrief struct {
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	CategoryURL  string `json:"category_url"`
}

type juejinAuthorInfo struct {
	UserName    string `json:"user_name"`
	Description string `json:"description"`
	AvatarLarge string `json:"avatar_large"`
}

type juejinTag struct {
	TagName string `json:"tag_name"`
}

type juejinArticleCategory struct {
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
}

type juejinArticleInfo struct {
	ArticleID    string `json:"article_id"`
	Title        string `json:"title"`
	BriefContent string `json:"brief_content"`
	Ctime        string `json:"ctime"` // unix seconds, JSON string
}

type juejinArticleEntry struct {
	ArticleID      string                `json:"article_id"`
	ArticleInfo    juejinArticleInfo     `json:"article_info"`
	AuthorUserInfo juejinAuthorInfo      `json:"author_user_info"`
	Category       juejinArticleCategory `json:"category"`
	Tags           []juejinTag           `json:"tags"`
}

// juejinFeedCard wraps entries in the recommend_all_feed payload.
type juejinFeedCard struct {
	ItemType int                `json:"item_type"`
	ItemInfo juejinArticleEntry `json:"item_info"`
}

func (r *juejinResp) ok() error {
	if r.ErrNo != 0 {
		return fmt.Errorf("juejin api error %d: %s", r.ErrNo, r.ErrMsg)
	}
	return nil
}

// fetchJuejinCategoryBriefs resolves category slugs to ids/names.
func fetchJuejinCategoryBriefs(c *ctxpkg.Context) ([]juejinCategoryBrief, error) {
	var resp juejinResp
	if err := juejinProfile.Fetch("https://api.juejin.cn/tag_api/v1/query_category_briefs").
		GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.ok(); err != nil {
		return nil, err
	}
	var briefs []juejinCategoryBrief
	if err := json.Unmarshal(resp.Data, &briefs); err != nil {
		return nil, err
	}
	return briefs, nil
}

// mapJuejinEntries converts article entries into feed items.
func mapJuejinEntries(feed *models.Feed, entries []juejinArticleEntry) {
	for _, e := range entries {
		info := e.ArticleInfo
		title := strings.TrimSpace(info.Title)
		link := "https://juejin.cn/post/" + info.ArticleID
		if title == "" || link == "" || info.ArticleID == "" {
			continue
		}
		desc := ""
		if brief := strings.TrimSpace(info.BriefContent); brief != "" {
			desc = "<p>" + html.EscapeString(brief) + "</p>"
		}
		pubDate := time.Time{}
		if sec, err := strconv.ParseInt(info.Ctime, 10, 64); err == nil && sec > 0 {
			pubDate = time.Unix(sec, 0)
		}
		item := routeutils.NewItem(title, link, desc, pubDate)
		item.GUID = info.ArticleID
		routeutils.SetItemAuthor(item, e.AuthorUserInfo.UserName, "", "")
		cats := make([]string, 0, len(e.Tags)+1)
		if e.Category.CategoryName != "" {
			cats = append(cats, e.Category.CategoryName)
		}
		for _, t := range e.Tags {
			if t.TagName != "" {
				cats = append(cats, t.TagName)
			}
		}
		routeutils.SetCategories(item, cats...)
		routeutils.AddItem(feed, item)
	}
}

// postJuejinArticles performs a POST against a recommend/content API and maps the entry list.
func postJuejinArticles(c *ctxpkg.Context, url string, body map[string]any, unwrapCards bool) ([]juejinArticleEntry, error) {
	var resp juejinResp
	if err := juejinProfile.PostJSON(url, body).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.ok(); err != nil {
		return nil, err
	}
	if !unwrapCards {
		var entries []juejinArticleEntry
		if err := json.Unmarshal(resp.Data, &entries); err != nil {
			return nil, err
		}
		return entries, nil
	}
	var cards []juejinFeedCard
	if err := json.Unmarshal(resp.Data, &cards); err != nil {
		return nil, err
	}
	entries := make([]juejinArticleEntry, 0, len(cards))
	for _, card := range cards {
		if card.ItemType == 2 && card.ItemInfo.ArticleID != "" {
			entries = append(entries, card.ItemInfo)
		}
	}
	return entries, nil
}

// --- Routes ---

var juejinCategoryRoute = routeutils.RouteSpec{
	Path:        "category/:category",
	Name:        "Juejin Category",
	Example:     "juejin/category/frontend",
	Maintainers: []string{"xihale"},
	Description: "Latest articles in one Juejin (稀土掘金) category",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("category", "category slug: backend, frontend, android, ios, ai, freebie, career, article"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  JuejinCategoryHandler,
}

// JuejinCategoryHandler handles /juejin/category/:category
func JuejinCategoryHandler(c *ctxpkg.Context) (*models.Feed, error) {
	category := c.Param("category")
	briefs, err := fetchJuejinCategoryBriefs(c)
	if err != nil {
		return nil, err
	}
	var cat *juejinCategoryBrief
	for i := range briefs {
		if briefs[i].CategoryURL == category {
			cat = &briefs[i]
			break
		}
	}
	if cat == nil {
		return nil, fmt.Errorf("juejin: unknown category %q", category)
	}

	entries, err := postJuejinArticles(c,
		"https://api.juejin.cn/recommend_api/v1/article/recommend_cate_feed",
		map[string]any{
			"id_type":   2,
			"sort_type": 300,
			"cate_id":   cat.CategoryID,
			"cursor":    "0",
			"limit":     20,
		}, false)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("掘金 "+cat.CategoryName, "https://juejin.cn/"+category, "掘金 "+cat.CategoryName+" 分类最新文章")
	mapJuejinEntries(feed, entries)
	return feed, nil
}

var juejinTrendingRoute = routeutils.RouteSpec{
	Path:        "trending/:category/:type",
	Name:        "Juejin Trending",
	Example:     "juejin/trending/all/weekly",
	Maintainers: []string{"xihale"},
	Description: "Trending Juejin articles by period (weekly/monthly/historical)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("category", "category slug or all"),
		routeutils.RequiredParam("type", "weekly, monthly or historical"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  JuejinTrendingHandler,
}

// JuejinTrendingHandler handles /juejin/trending/:category/:type
func JuejinTrendingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	category := c.Param("category")
	typ := routeutils.ParseEnum(c.Param("type"), "weekly", "weekly", "monthly", "historical")

	type trendingParams struct {
		title    string
		link     string
		sortType int
	}
	params := map[string]trendingParams{
		"monthly":    {title: "本月", link: "monthly_hottest", sortType: 30},
		"weekly":     {title: "本周", link: "weekly_hottest", sortType: 7},
		"historical": {title: "历史", link: "hottest", sortType: 0},
	}
	p := params[typ]

	name := ""
	id := ""
	briefs, err := fetchJuejinCategoryBriefs(c)
	if err != nil {
		return nil, err
	}
	for _, b := range briefs {
		if b.CategoryURL == category {
			id = b.CategoryID
			name = b.CategoryName
			break
		}
	}
	if name == "" && category != "all" {
		return nil, fmt.Errorf("juejin: unknown category %q", category)
	}
	if name == "" {
		name = "全部"
	}

	body := map[string]any{
		"cursor":    "0",
		"id_type":   2,
		"limit":     20,
		"sort_type": p.sortType,
	}
	unwrapCards := false
	endpoint := "https://api.juejin.cn/recommend_api/v1/article/recommend_cate_feed"
	if id == "" {
		// all-category trending comes from the global feed of wrapped cards
		endpoint = "https://api.juejin.cn/recommend_api/v1/article/recommend_all_feed"
		unwrapCards = true
	} else {
		body["cate_id"] = id
	}
	entries, err := postJuejinArticles(c, endpoint, body, unwrapCards)
	if err != nil {
		return nil, err
	}

	fallbackURL := category
	if fallbackURL == "all" || id == "" {
		fallbackURL = "recommended"
	}
	feed := routeutils.NewFeed(
		fmt.Sprintf("掘金%s%s最热", name, p.title),
		fmt.Sprintf("https://juejin.cn/%s?sort=%s", fallbackURL, p.link),
		"掘金热门文章",
	)
	mapJuejinEntries(feed, entries)
	return feed, nil
}

var juejinPostsRoute = routeutils.RouteSpec{
	Path:        "posts/:id",
	Name:        "Juejin User Posts",
	Example:     "juejin/posts/3051900006845944",
	Maintainers: []string{"xihale"},
	Description: "Latest posts by a Juejin user",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "user id, found in the user page URL"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  JuejinPostsHandler,
}

// JuejinPostsHandler handles /juejin/posts/:id
func JuejinPostsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	entries, err := postJuejinArticles(c,
		"https://api.juejin.cn/content_api/v1/article/query_list",
		map[string]any{
			"user_id":   id,
			"sort_type": 2,
			"cursor":    "0",
			"limit":     20,
		}, false)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("juejin: no posts found for user %q", id)
	}

	author := entries[0].AuthorUserInfo
	feed := routeutils.NewFeedWithOptions(routeutils.FeedOptions{
		Title:       "掘金专栏 - " + author.UserName,
		Link:        "https://juejin.cn/user/" + id + "/posts",
		Description: author.Description,
		Image:       author.AvatarLarge,
	})
	mapJuejinEntries(feed, entries)
	return feed, nil
}
