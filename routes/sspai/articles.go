package routes

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const (
	sspaiBaseURL   = "https://sspai.com"
	sspaiAPIPrefix = "https://sspai.com/api/v1"
	sspaiCDNPrefix = "https://cdnfile.sspai.com/"
	// sspaiTagHost is the legacy tag page host required as Referer by the tag API.
	sspaiTagHost = "https://beta.sspai.com"
)

// sspaiAPIProfile returns a fresh JSON profile; callers may chain Referer()
// because Profile modifiers mutate the receiver in place.
func sspaiAPIProfile() *disguise.Profile {
	return disguise.Chrome().JSONAccept().Lang("zh-CN,zh;q=0.9")
}

// --- Upstream payload types (sspai.com/api/v1) ---

type sspaiAuthor struct {
	ID       int64  `json:"id"`
	Nickname string `json:"nickname"`
	Slug     string `json:"slug"`
}

type sspaiArticle struct {
	ID           int64       `json:"id"`
	Title        string      `json:"title"`
	Summary      string      `json:"summary"`
	Banner       string      `json:"banner"`
	ReleasedAt   int64       `json:"released_at"`   // /articles payload
	ReleasedTime int64       `json:"released_time"` // index/favorites payload
	Author       sspaiAuthor `json:"author"`
}

// publishedAt unifies the two timestamp field spellings used by sspai APIs.
func (a sspaiArticle) publishedAt() int64 {
	if a.ReleasedAt > 0 {
		return a.ReleasedAt
	}
	return a.ReleasedTime
}

type sspaiListResp struct {
	List []sspaiArticle `json:"list"`
}

// sspaiWrapped is the {error, msg, data} envelope of page/get style endpoints.
type sspaiWrapped[T any] struct {
	Error int    `json:"error"`
	Msg   string `json:"msg"`
	Data  T      `json:"data"`
}

type sspaiSpecialColumn struct {
	Title string `json:"title"`
	Intro string `json:"intro"`
}

type sspaiTopic struct {
	ID         int64       `json:"id"`
	Title      string      `json:"title"`
	Intro      string      `json:"intro"`
	Banner     string      `json:"banner"`
	ReleasedAt int64       `json:"released_at"`
	Author     sspaiAuthor `json:"author"`
}

// fetchSspaiArticles queries /api/v1/articles with the given query parameters.
// offset/include_total defaults are applied when absent.
func fetchSspaiArticles(c *ctxpkg.Context, referer string, query url.Values) ([]sspaiArticle, error) {
	if query.Get("offset") == "" {
		query.Set("offset", "0")
	}
	query.Set("include_total", "false")
	endpoint := sspaiAPIPrefix + "/articles?" + query.Encode()

	var resp sspaiListResp
	if err := sspaiAPIProfile().Referer(referer).Fetch(endpoint).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	return resp.List, nil
}

// sspaiArticleLink builds the public article URL for an id.
func sspaiArticleLink(id int64) string {
	return fmt.Sprintf("%s/post/%d", sspaiBaseURL, id)
}

// mapSspaiArticles converts article summaries into feed items.
// guidPrefix namespaces item GUIDs per feed kind (e.g. "sspai-matrix-").
func mapSspaiArticles(feed *models.Feed, articles []sspaiArticle, guidPrefix string) {
	for _, art := range articles {
		title := strings.TrimSpace(art.Title)
		if title == "" || art.ID == 0 {
			continue
		}
		desc := ""
		if summary := strings.TrimSpace(art.Summary); summary != "" {
			desc = "<p>" + html.EscapeString(summary) + "</p>"
		}
		pubDate := time.Time{}
		if sec := art.publishedAt(); sec > 0 {
			pubDate = time.Unix(sec, 0)
		}
		item := routeutils.NewItem(title, sspaiArticleLink(art.ID), desc, pubDate)
		item.GUID = guidPrefix + strconv.FormatInt(art.ID, 10)
		routeutils.SetItemAuthor(item, art.Author.Nickname, "", "")
		routeutils.AddItem(feed, item)
	}
}

// --- Routes ---

var sspaiMatrixRoute = routeutils.RouteSpec{
	Path:        "matrix",
	Name:        "sspai Matrix",
	Example:     "sspai/matrix",
	Maintainers: []string{"xihale"},
	Description: "少数派 Matrix 平板专栏文章",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     SspaiMatrixHandler,
}

// SspaiMatrixHandler handles /sspai/matrix
func SspaiMatrixHandler(c *ctxpkg.Context) (*models.Feed, error) {
	articles, err := fetchSspaiArticles(c, sspaiBaseURL+"/", url.Values{
		"limit":     {"20"},
		"is_matrix": {"1"},
		"sort":      {"matrix_at"},
	})
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("少数派 -- Matrix", sspaiBaseURL+"/matrix", "少数派 -- Matrix")
	mapSspaiArticles(feed, articles, "sspai-matrix-")
	return feed, nil
}

var sspaiIndexRoute = routeutils.RouteSpec{
	Path:        "index",
	Name:        "sspai Home",
	Example:     "sspai/index",
	Maintainers: []string{"xihale"},
	Description: "少数派首页资讯流",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     SspaiIndexHandler,
}

// SspaiIndexHandler handles /sspai/index
func SspaiIndexHandler(c *ctxpkg.Context) (*models.Feed, error) {
	endpoint := sspaiAPIPrefix + "/article/index/page/get?limit=10&offset=0&created_at=0"
	resp, err := fetchSspaiWrapped[[]sspaiArticle](c, endpoint, sspaiBaseURL+"/")
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("少数派", sspaiBaseURL+"/", "少数派首页")
	mapSspaiArticles(feed, resp.Data, "sspai-index-")
	return feed, nil
}

var sspaiColumnRoute = routeutils.RouteSpec{
	Path:        "column/:id",
	Name:        "sspai Column",
	Example:     "sspai/column/262",
	Maintainers: []string{"xihale"},
	Description: "少数派付费/免费专栏文章列表（不含付费正文）",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "专栏 id，可在专栏页 URL 中找到"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  SspaiColumnHandler,
}

// SspaiColumnHandler handles /sspai/column/:id
func SspaiColumnHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	referer := sspaiBaseURL + "/column/" + id

	colResp, err := fetchSspaiWrapped[sspaiSpecialColumn](c, sspaiAPIPrefix+"/special_columns/"+url.PathEscape(id), referer)
	if err != nil {
		return nil, err
	}

	articles, err := fetchSspaiArticles(c, referer, url.Values{
		"limit":              {"10"},
		"special_column_ids": {id},
	})
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(colResp.Data.Title)
	if title == "" {
		title = "专栏 " + id
	}
	feed := routeutils.NewFeed("少数派专栏-"+title, referer, colResp.Data.Intro)
	mapSspaiArticles(feed, articles, "sspai-column-")
	return feed, nil
}

var sspaiAuthorRoute = routeutils.RouteSpec{
	Path:        "author/:id",
	Name:        "sspai Author",
	Example:     "sspai/author/796518",
	Maintainers: []string{"xihale"},
	Description: "少数派作者文章动态",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "作者 slug 或数字 id，slug 可在作者主页 URL 中找到"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  SspaiAuthorHandler,
}

type sspaiSlugInfo struct {
	ID       int64  `json:"id"`
	Nickname string `json:"nickname"`
}

// resolveSspaiUserID maps a numeric id through unchanged and resolves a slug
// via user/slug/info/get; returns an error for unknown users.
func resolveSspaiUserID(c *ctxpkg.Context, raw string) (int64, error) {
	if isSSPAINumeric(raw) {
		return strconv.ParseInt(raw, 10, 64)
	}
	referer := fmt.Sprintf("%s/u/%s/posts", sspaiBaseURL, raw)
	resp, err := fetchSspaiWrapped[sspaiSlugInfo](c, sspaiAPIPrefix+"/user/slug/info/get?slug="+url.QueryEscape(raw), referer)
	if err != nil {
		return 0, err
	}
	if resp.Error != 0 || resp.Data.ID == 0 {
		return 0, fmt.Errorf("sspai: user not found: %s", raw)
	}
	return resp.Data.ID, nil
}

// isSSPAINumeric reports whether s consists of ASCII digits only, mirroring
// the upstream /^\d+$/ check that separates numeric ids from slugs.
func isSSPAINumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// SspaiAuthorHandler handles /sspai/author/:id
func SspaiAuthorHandler(c *ctxpkg.Context) (*models.Feed, error) {
	raw := c.Param("id")
	authorID, err := resolveSspaiUserID(c, raw)
	if err != nil {
		return nil, err
	}

	articles, err := fetchSspaiArticles(c, sspaiBaseURL+"/", url.Values{
		"limit":      {"20"},
		"author_ids": {strconv.FormatInt(authorID, 10)},
	})
	if err != nil {
		return nil, err
	}
	if len(articles) == 0 {
		return nil, fmt.Errorf("sspai: no articles found for author %q", raw)
	}

	nickname := articles[0].Author.Nickname
	slug := articles[0].Author.Slug
	if nickname == "" {
		nickname = raw
	}
	link := sspaiBaseURL + "/u/"
	if slug != "" {
		link += slug
	} else {
		link += strconv.FormatInt(authorID, 10)
	}
	link += "/posts"

	feed := routeutils.NewFeed(nickname+" - 少数派作者", link, nickname+" 更新推送")
	mapSspaiArticles(feed, articles, "sspai-author-")
	return feed, nil
}

var sspaiTagRoute = routeutils.RouteSpec{
	Path:        "tag/:keyword",
	Name:        "sspai Tag",
	Example:     "sspai/tag/apple",
	Maintainers: []string{"xihale"},
	Description: "少数派标签下最新文章",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("keyword", "关键词，可在标签页 URL 中找到"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  SspaiTagHandler,
}

// SspaiTagHandler handles /sspai/tag/:keyword
func SspaiTagHandler(c *ctxpkg.Context) (*models.Feed, error) {
	keyword := c.Param("keyword")
	encoded := url.QueryEscape(keyword)
	host := sspaiTagHost + "/tag/" + encoded

	articles, err := fetchSspaiArticles(c, host, url.Values{
		"limit":   {"50"},
		"has_tag": {"1"},
		"tag":     {encoded},
	})
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("#"+keyword+" - 少数派", host, keyword+" 更新推送")
	mapSspaiArticles(feed, articles, "sspai-tag-")
	return feed, nil
}

// fetchSspaiWrapped performs a GET against a wrapped {error,msg,data} endpoint.
func fetchSspaiWrapped[T any](c *ctxpkg.Context, endpoint, referer string) (*sspaiWrapped[T], error) {
	var resp sspaiWrapped[T]
	if err := sspaiAPIProfile().Referer(referer).Fetch(endpoint).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Error != 0 {
		return nil, fmt.Errorf("sspai api error %d (%s): %s", resp.Error, resp.Msg, endpoint)
	}
	return &resp, nil
}
