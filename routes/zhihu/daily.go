package routes

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/client"
	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

// ---------------- 知乎热榜 /zhihu/hot ----------------

type zhihuHotItem struct {
	DetailText string `json:"detail_text"`
	Target     struct {
		ID          zhihuInt64 `json:"id"`
		Title       string     `json:"title"`
		Created     int64      `json:"created"`
		Excerpt     string     `json:"excerpt"`
		AnswerCount int        `json:"answer_count"`
	} `json:"target"`
}

type zhihuHotResp struct {
	Data []zhihuHotItem `json:"data"`
}

func questionLink(id zhihuInt64) string {
	return fmt.Sprintf("https://www.zhihu.com/question/%d", id)
}

var zhihuHotRoute = routeutils.RouteSpec{
	Path:        "hot",
	Name:        "知乎热榜",
	Example:     "zhihu/hot",
	Maintainers: []string{"xihale"},
	Description: "知乎全站热榜",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "条数，默认 30，上限 50"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  ZhihuHotHandler,
}

// ZhihuHotHandler handles /zhihu/hot
func ZhihuHotHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 50)
	ctx := c.Parent()

	apiURL := fmt.Sprintf("https://api.zhihu.com/topstory/hot-lists/total?limit=%d&reverse_order=0", limit)
	var resp zhihuHotResp
	if err := zhihuProfile("https://www.zhihu.com/hot").Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("知乎热榜", "https://www.zhihu.com/hot", "知乎全站热门内容")
	routeutils.AppendMappedItems(feed, resp.Data, 0, func(entry zhihuHotItem) *models.Item {
		t := entry.Target
		if t.Title == "" || t.ID == 0 {
			return nil
		}
		desc := ""
		if t.Excerpt != "" {
			desc = "<p>" + html.EscapeString(t.Excerpt) + "</p>"
		}
		meta := fmt.Sprintf("回答数：%d", t.AnswerCount)
		if entry.DetailText != "" {
			meta += " | 热度：" + html.EscapeString(entry.DetailText)
		}
		desc += "<p>" + meta + "</p>"

		item := routeutils.NewItem(t.Title, questionLink(t.ID), desc, time.Unix(t.Created, 0))
		item.GUID = fmt.Sprintf("zhihu-question-%d", t.ID)
		return item
	})
	return feed, nil
}

// ---------------- 知乎日报 /zhihu/daily 与合集 ----------------

const zhihuDailyAPI = "https://daily.zhihu.com/api/4"

type zhihuDailyStoryRef struct {
	ID    zhihuInt64 `json:"id"`
	Title string     `json:"title"`
	URL   string     `json:"url"`
}

type zhihuDailyLatest struct {
	Date    string               `json:"date"` // YYYYMMDD
	Stories []zhihuDailyStoryRef `json:"stories"`
}

type zhihuStory struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	URL         string `json:"url"`
	PublishTime string `json:"publish_time"` // unix seconds as string, often absent
}

// publishTimeOr prefers the story's own unix timestamp and falls back to the
// edition date of the daily issue it appeared in.
func (s zhihuStory) publishTimeOr(fallback time.Time) time.Time {
	if s.PublishTime != "" {
		if sec, err := strconv.ParseInt(s.PublishTime, 10, 64); err == nil && sec > 0 {
			return time.Unix(sec, 0)
		}
	}
	return fallback
}

func parseZhihuYYYYMMDD(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	if t, err := time.ParseInLocation("20060102", raw, time.Local); err == nil {
		return t
	}
	return time.Time{}
}

func fetchZhihuStory(ctx context.Context, cl *client.Client, apiURL string) (zhihuStory, error) {
	var story zhihuStory
	err := zhihuProfile("https://daily.zhihu.com/").Fetch(apiURL).GetJSON(ctx, cl, &story)
	return story, err
}

var zhihuDailyRoute = routeutils.RouteSpec{
	Path:        "daily",
	Name:        "知乎日报",
	Example:     "zhihu/daily",
	Maintainers: []string{"xihale"},
	Description: "知乎日报每日精选，含全文",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     ZhihuDailyHandler,
}

// ZhihuDailyHandler handles /zhihu/daily
func ZhihuDailyHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()

	var latest zhihuDailyLatest
	if err := zhihuProfile("https://daily.zhihu.com/").Fetch(zhihuDailyAPI+"/news/latest").GetJSON(ctx, c.Client(), &latest); err != nil {
		return nil, err
	}

	editionDate := parseZhihuYYYYMMDD(latest.Date)
	feed := routeutils.NewFeed("知乎日报", "https://daily.zhihu.com/", "每天3次，每次7分钟")
	routeutils.AppendMappedItems(feed, latest.Stories, 0, func(s zhihuDailyStoryRef) *models.Item {
		story, err := fetchZhihuStory(ctx, c.Client(), zhihuDailyAPI+"/news/"+s.ID.String())
		if err != nil || story.Title == "" {
			// Detail unavailable: still emit list metadata with edition date.
			return routeutils.NewItem(s.Title, s.URL, "", editionDate)
		}
		link := story.URL
		if link == "" {
			link = s.URL
		}
		return routeutils.NewItem(story.Title, link, processZhihuContent(story.Body), story.publishTimeOr(editionDate))
	})
	return feed, nil
}

// ---------------- 知乎日报合集 /zhihu/daily/section/:sectionId ----------------

type zhihuSectionStory struct {
	ID          zhihuInt64 `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Date        string     `json:"date"` // YYYYMMDD
	DisplayDate string     `json:"display_date"`
}

type zhihuSectionResp struct {
	Name    string              `json:"name"`
	Stories []zhihuSectionStory `json:"stories"`
}

var zhihuDailySectionRoute = routeutils.RouteSpec{
	Path:        "daily/section/:sectionId",
	Name:        "知乎日报 - 合集",
	Example:     "zhihu/daily/section/2",
	Maintainers: []string{"xihale"},
	Description: "知乎日报主题合集",
	Categories:  []models.Category{{Name: "new-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("sectionId", "合集 id，完整列表见 news-at.zhihu.com/api/7/sections"),
		routeutils.OptionalParam("limit", "条数，默认 20，上限 100（每条需请求一次详情）"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  ZhihuDailySectionHandler,
}

// ZhihuDailySectionHandler handles /zhihu/daily/section/:sectionId
func ZhihuDailySectionHandler(c *ctxpkg.Context) (*models.Feed, error) {
	sectionID := c.Param("sectionId")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 100)
	ctx := c.Parent()

	apiBase := "https://news-at.zhihu.com/api/7"
	var resp zhihuSectionResp
	if err := zhihuProfile(apiBase+"/section/"+sectionID).
		Fetch(apiBase+"/section/"+sectionID).
		GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}

	title := resp.Name
	if title == "" {
		title = "合集 " + sectionID
	}
	feed := routeutils.NewFeed(title+" - 知乎日报", "https://daily.zhihu.com/", "每天3次，每次7分钟")

	for _, s := range resp.Stories {
		if len(feed.Items) >= limit {
			break
		}
		// 过滤极个别站外链接（与上游一致）
		if !strings.HasPrefix(s.URL, "https://daily.zhihu.com/") {
			continue
		}
		story, err := fetchZhihuStory(ctx, c.Client(), apiBase+"/news/"+s.ID.String())
		if err != nil || story.Title == "" {
			item := routeutils.NewItem(s.Title, s.URL, "", parseZhihuYYYYMMDD(s.Date))
			if item != nil {
				feed.Items = append(feed.Items, *item)
			}
			continue
		}
		pub := story.publishTimeOr(parseZhihuYYYYMMDD(s.Date))
		if item := routeutils.NewItem(story.Title, story.URL, processZhihuContent(story.Body), pub); item != nil {
			feed.Items = append(feed.Items, *item)
		}
	}
	return feed, nil
}
