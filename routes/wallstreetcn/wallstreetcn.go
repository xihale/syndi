package routes

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// wscProfile disguises requests against the wallstreetcn APIs.
var wscProfile = disguise.Chrome().JSONAccept().Referer("https://wallstreetcn.com/")

const (
	wscAPIHost   = "https://api-one.wallstcn.com"
	wscHotAPIURL = "https://api-one-wscn.awtmt.com"
	wscRootURL   = "https://wallstreetcn.com"
)

// --- Upstream payload types ---

type wscResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (r *wscResp) ok() error {
	if r.Code != 20000 {
		return fmt.Errorf("wallstreetcn api error %d: %s", r.Code, r.Message)
	}
	return nil
}

type wscAuthor struct {
	DisplayName string `json:"display_name"`
}

type wscImage struct {
	URI    string `json:"uri"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type wscAssetTag struct {
	Name string `json:"name"`
}

// wscLive is one entry of /apiv1/content/lives.
type wscLive struct {
	ID          int64         `json:"id"`
	Title       string        `json:"title"`
	ContentText string        `json:"content_text"`
	Content     string        `json:"content"`
	ContentMore string        `json:"content_more"`
	DisplayTime int64         `json:"display_time"` // unix seconds
	URI         string        `json:"uri"`
	Score       int           `json:"score"`
	Author      *wscAuthor    `json:"author"`
	Images      []wscImage    `json:"images"`
	Tags        []wscAssetTag `json:"tags"`
}

// wscResource is the embedded payload inside information-flow entries.
type wscResource struct {
	ID          int64      `json:"id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	ContentText string     `json:"content_text"`
	Content     string     `json:"content"`
	ContentMore string     `json:"content_more"`
	DisplayTime int64      `json:"display_time"`
	URI         string     `json:"uri"`
	Author      *wscAuthor `json:"author"`
}

type wscFlowItem struct {
	ResourceType string      `json:"resource_type"`
	Resource     wscResource `json:"resource"`
}

// wscArticleDetail is the payload of /apiv1/content/articles/:id.
type wscArticleDetail struct {
	ID          int64         `json:"id"`
	Title       string        `json:"title"`
	ContentText string        `json:"content_text"`
	Content     string        `json:"content"`
	ContentMore string        `json:"content_more"`
	DisplayTime int64         `json:"display_time"`
	SourceName  string        `json:"source_name"`
	Author      *wscAuthor    `json:"author"`
	AssetTags   []wscAssetTag `json:"asset_tags"`
}

type wscHotItem struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	URI         string `json:"uri"`
	DisplayTime int64  `json:"display_time"`
}

func wscTitleOrText(title, text string) string {
	if t := strings.TrimSpace(title); t != "" {
		return t
	}
	return strings.TrimSpace(text)
}

// wscFetchJSON performs a disguised GET returning the envelope.
func wscFetchJSON(c *ctxpkg.Context, url string) (*wscResp, error) {
	var resp wscResp
	if err := wscProfile.Fetch(url).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.ok(); err != nil {
		return nil, err
	}
	return &resp, nil
}

// wscFetchArticle loads one article detail; returns nil when deleted (60301).
func wscFetchArticle(c *ctxpkg.Context, apiHost string, id int64) (*wscArticleDetail, error) {
	var resp wscResp
	url := fmt.Sprintf("%s/apiv1/content/articles/%d?extract=0", apiHost, id)
	if err := wscProfile.Fetch(url).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 20000 {
		return nil, nil // content missing or deleted
	}
	var detail wscArticleDetail
	if err := json.Unmarshal(resp.Data, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// wscFetchArticlesConcurrent loads article details with bounded parallelism,
// preserving input order. Deleted articles yield nil entries.
func wscFetchArticlesConcurrent(c *ctxpkg.Context, apiHost string, ids []int64, concurrency int) ([]*wscArticleDetail, error) {
	out := make([]*wscArticleDetail, len(ids))
	if concurrency < 1 {
		concurrency = 4
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex
	for i, id := range ids {
		wg.Add(1)
		go func(idx int, aid int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			detail, err := wscFetchArticle(c, apiHost, aid)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			out[idx] = detail
		}(i, id)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func wscFeedImage() string { return "https://static.wscn.net/wscn/_static/favicon.png" }

var wscLiveRoute = routeutils.RouteSpec{
	Path:        "live",
	Name:        "Wallstreetcn Live News",
	Example:     "wallstreetcn/live",
	Maintainers: []string{"xihale"},
	Description: "Wallstreetcn (华尔街见闻) real-time live news flash",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    5 * time.Minute,
	Handler:     WscLiveHandler,
}

var wscLiveCategoryRoute = routeutils.RouteSpec{
	Path:        "live/:category",
	Name:        "Wallstreetcn Live News By Category",
	Example:     "wallstreetcn/live/a-stock",
	Maintainers: []string{"xihale"},
	Description: "Wallstreetcn (华尔街见闻) live news by channel",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("category", "global 要闻, a-stock A股, us-stock 美股, hk-stock 港股, forex 外汇, commodity 商品 or financing 理财"),
	},
	CacheTTL: 5 * time.Minute,
	Handler:  WscLiveHandler,
}

// WscLiveHandler handles /wallstreetcn/live/:category?
func WscLiveHandler(c *ctxpkg.Context) (*models.Feed, error) {
	titles := map[string]string{
		"global": "要闻", "a-stock": "A股", "us-stock": "美股", "hk-stock": "港股",
		"forex": "外汇", "commodity": "商品", "financing": "理财",
	}
	category := c.Param("category")
	if category == "" {
		category = "global"
	}
	title, ok := titles[category]
	if !ok {
		return nil, fmt.Errorf("wallstreetcn: unknown live category %q", category)
	}

	resp, err := wscFetchJSON(c, fmt.Sprintf("%s/apiv1/content/lives?channel=%s-channel&limit=100", wscAPIHost, category))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Items []wscLive `json:"items"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeedWithOptions(routeutils.FeedOptions{
		Title:       fmt.Sprintf("华尔街见闻 - 实时快讯 - %s", title),
		Link:        fmt.Sprintf("%s/live/%s", wscRootURL, category),
		Description: "华尔街见闻实时快讯",
		Image:       wscFeedImage(),
	})
	for _, l := range payload.Items {
		liveTitle := wscTitleOrText(l.Title, l.ContentText)
		if liveTitle == "" || l.URI == "" {
			continue
		}
		desc := l.Content + l.ContentMore
		for _, img := range l.Images {
			if img.URI != "" {
				desc += fmt.Sprintf(`<img src="%s" width="%d" height="%d"/>`, img.URI, img.Width, img.Height)
			}
		}
		pubDate := time.Time{}
		if l.DisplayTime > 0 {
			pubDate = time.Unix(l.DisplayTime, 0)
		}
		item := routeutils.NewItem(liveTitle, l.URI, desc, pubDate)
		item.GUID = fmt.Sprintf("%d", l.ID)
		author := ""
		if l.Author != nil {
			author = l.Author.DisplayName
		}
		routeutils.SetItemAuthor(item, author, "", "")
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

var wscNewsRoute = routeutils.RouteSpec{
	Path:        "news",
	Name:        "Wallstreetcn News",
	Example:     "wallstreetcn/news",
	Maintainers: []string{"xihale"},
	Description: "Wallstreetcn (华尔街见闻) latest news feed",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     WscNewsHandler,
}

var wscNewsCategoryRoute = routeutils.RouteSpec{
	Path:        "news/:category",
	Name:        "Wallstreetcn News By Category",
	Example:     "wallstreetcn/news/shares",
	Maintainers: []string{"xihale"},
	Description: "Wallstreetcn (华尔街见闻) news by channel",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("category", "global 最新, shares 股市, bonds 债市, commodities 商品, forex 外汇, enterprise 公司, asset-manage 资管, tmt 科技, estate 地产, car 汽车 or medicine 医药"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  WscNewsHandler,
}

// WscNewsHandler handles /wallstreetcn/news/:category?
func WscNewsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	titles := map[string]string{
		"global": "最新", "shares": "股市", "bonds": "债市", "commodities": "商品",
		"forex": "外汇", "enterprise": "公司", "asset-manage": "资管", "tmt": "科技",
		"estate": "地产", "car": "汽车", "medicine": "医药",
	}
	category := c.Param("category")
	if category == "" {
		category = "global"
	}
	title, ok := titles[category]
	if !ok {
		return nil, fmt.Errorf("wallstreetcn: unknown news category %q", category)
	}

	resp, err := wscFetchJSON(c, fmt.Sprintf("%s/apiv1/content/information-flow?channel=%s-channel&accept=article&limit=20", wscAPIHost, category))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Items []wscFlowItem `json:"items"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		return nil, err
	}

	// Collect non-ad entries; remember which need article detail fetches.
	type entry struct {
		res    wscResource
		detail *wscArticleDetail
	}
	entries := make([]entry, 0, len(payload.Items))
	var needIDs []int64
	var needIdx []int
	for _, it := range payload.Items {
		if it.ResourceType == "ad" || it.Resource.ID == 0 {
			continue
		}
		if len(entries) >= 20 {
			break
		}
		if it.ResourceType == "article" {
			needIDs = append(needIDs, it.Resource.ID)
			needIdx = append(needIdx, len(entries))
		}
		entries = append(entries, entry{res: it.Resource})
	}
	if len(needIDs) > 0 {
		details, err := wscFetchArticlesConcurrent(c, wscAPIHost, needIDs, 4)
		if err != nil {
			return nil, err
		}
		for i, d := range details {
			entries[needIdx[i]].detail = d
		}
	}

	feed := routeutils.NewFeedWithOptions(routeutils.FeedOptions{
		Title:       fmt.Sprintf("华尔街见闻 - 资讯 - %s", title),
		Link:        fmt.Sprintf("%s/news/%s", wscRootURL, category),
		Description: "华尔街见闻资讯",
		Image:       wscFeedImage(),
	})
	for _, e := range entries {
		res := e.res
		link := res.URI
		itemTitle := wscTitleOrText(res.Title, res.ContentText)
		desc := res.Content + res.ContentMore
		author := ""
		if res.Author != nil {
			author = res.Author.DisplayName
		}
		cats := []string{}
		displayTime := res.DisplayTime
		guid := fmt.Sprintf("%d", res.ID)
		if e.detail != nil {
			itemTitle = wscTitleOrText(e.detail.Title, e.detail.ContentText)
			desc = e.detail.Content + e.detail.ContentMore
			if e.detail.SourceName != "" {
				author = e.detail.SourceName
			} else if e.detail.Author != nil {
				author = e.detail.Author.DisplayName
			}
			for _, t := range e.detail.AssetTags {
				cats = append(cats, t.Name)
			}
			if e.detail.DisplayTime > 0 {
				displayTime = e.detail.DisplayTime
			}
		}
		if itemTitle == "" || link == "" {
			continue
		}
		pubDate := time.Time{}
		if displayTime > 0 {
			pubDate = time.Unix(displayTime, 0)
		}
		item := routeutils.NewItem(itemTitle, link, desc, pubDate)
		item.GUID = guid
		routeutils.SetItemAuthor(item, author, "", "")
		routeutils.SetCategories(item, cats...)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

var wscHotRoute = routeutils.RouteSpec{
	Path:        "hot",
	Name:        "Wallstreetcn Hot Articles",
	Example:     "wallstreetcn/hot",
	Maintainers: []string{"xihale"},
	Description: "Wallstreetcn (华尔街见闻) hottest articles of the day",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     WscHotHandler,
}

var wscHotPeriodRoute = routeutils.RouteSpec{
	Path:        "hot/:period",
	Name:        "Wallstreetcn Hot Articles By Period",
	Example:     "wallstreetcn/hot/week",
	Maintainers: []string{"xihale"},
	Description: "Wallstreetcn (华尔街见闻) hottest articles of the day or week",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("period", "day 当日 or week 当周"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  WscHotHandler,
}

// WscHotHandler handles /wallstreetcn/hot/:period?
func WscHotHandler(c *ctxpkg.Context) (*models.Feed, error) {
	period := routeutils.ParseEnum(c.Param("period"), "day", "day", "week")
	key := period + "_items"

	resp, err := wscFetchJSON(c, wscHotAPIURL+"/apiv1/content/articles/hot?period=all")
	if err != nil {
		return nil, err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		return nil, err
	}
	rawItems, ok := payload[key]
	if !ok {
		return nil, fmt.Errorf("wallstreetcn: missing %q in hot articles response", key)
	}
	var hotItems []wscHotItem
	if err := json.Unmarshal(rawItems, &hotItems); err != nil {
		return nil, err
	}
	if len(hotItems) > 20 {
		hotItems = hotItems[:20]
	}

	ids := make([]int64, len(hotItems))
	for i, h := range hotItems {
		ids[i] = h.ID
	}
	details, err := wscFetchArticlesConcurrent(c, wscHotAPIURL, ids, 4)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeedWithOptions(routeutils.FeedOptions{
		Title:       "华尔街见闻 - 最热文章",
		Link:        wscRootURL,
		Description: "华尔街见闻最热文章",
		Image:       wscFeedImage(),
	})
	for i, h := range hotItems {
		if h.URI == "" {
			continue
		}
		title := h.Title
		desc := ""
		author := ""
		cats := []string{}
		if details[i] != nil {
			title = wscTitleOrText(details[i].Title, title)
			desc = details[i].Content + details[i].ContentMore
			if details[i].SourceName != "" {
				author = details[i].SourceName
			} else if details[i].Author != nil {
				author = details[i].Author.DisplayName
			}
			for _, t := range details[i].AssetTags {
				cats = append(cats, t.Name)
			}
		}
		if title == "" {
			continue
		}
		pubDate := time.Time{}
		if h.DisplayTime > 0 {
			pubDate = time.Unix(h.DisplayTime, 0)
		}
		item := routeutils.NewItem(title, h.URI, desc, pubDate)
		item.GUID = fmt.Sprintf("%d", h.ID)
		routeutils.SetItemAuthor(item, author, "", "")
		routeutils.SetCategories(item, cats...)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
