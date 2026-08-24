package routes

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

const zhihuAPIBase = "https://www.zhihu.com/api/v4"

// ---------------- 知乎专栏 /zhihu/zhuanlan/:id ----------------

type zhihuColumnMeta struct {
	Title    string `json:"title"`
	ImageURL string `json:"image_url"`
	ID       string `json:"id"`
}

type zhihuColumnItem struct {
	Type        string     `json:"type"` // article | answer | zvideo
	ID          zhihuInt64 `json:"id"`
	Content     string     `json:"content"`
	Excerpt     string     `json:"excerpt"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Created     int64      `json:"created"`      // article
	CreatedTime int64      `json:"created_time"` // answer
	CreatedAt   int64      `json:"created_at"`   // zvideo
	Description string     `json:"description"`  // zvideo
	Author      struct {
		Name string `json:"name"`
	} `json:"author"`
	Question struct {
		ID    zhihuInt64 `json:"id"`
		Title string     `json:"title"`
	} `json:"question"`
}

type zhihuColumnItemsResp struct {
	Data []zhihuColumnItem `json:"data"`
}

var zhihuZhuanlanRoute = routeutils.RouteSpec{
	Path:        "zhuanlan/:id",
	Name:        "知乎专栏",
	Example:     "zhihu/zhuanlan/googledevelopers",
	Maintainers: []string{"xihale"},
	Description: "知乎专栏最新文章，支持新旧两种专栏 id",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true, EnvDeps: []string{zhihuCookiesEnv}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "专栏 id，可在专栏主页 URL 中找到（旧格式如 googledevelopers，新格式以 c_ 开头）"),
		routeutils.OptionalParam("limit", "条数，默认 20，上限 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  ZhihuZhuanlanHandler,
}

// ZhihuZhuanlanHandler handles /zhihu/zhuanlan/:id
func ZhihuZhuanlanHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()
	if err := requireZhihuCookies(); err != nil {
		return nil, err
	}

	columnURL := columnHomeLink(id)
	var meta zhihuColumnMeta
	if err := zhihuProfile(columnURL).Fetch(zhihuAPIBase+"/columns/"+id).GetJSON(ctx, c.Client(), &meta); err != nil {
		return nil, fmt.Errorf("获取专栏信息失败（请确认专栏 id 正确且 %s 有效）: %w", zhihuCookiesEnv, err)
	}

	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	apiURL := zhihuAPIBase + "/columns/" + id + "/items?" + q.Encode()
	var resp zhihuColumnItemsResp
	if err := zhihuProfile(columnURL).Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}

	title := meta.Title
	if title == "" {
		title = id
	}
	feed := routeutils.NewFeed("知乎专栏 - "+title, columnURL, "知乎专栏 "+id)
	routeutils.AppendMappedItems(feed, resp.Data, 0, mapZhihuColumnItem)
	return feed, nil
}

func columnHomeLink(id string) string {
	if strings.HasPrefix(id, "c_") {
		return "https://www.zhihu.com/column/" + id
	}
	return "https://zhuanlan.zhihu.com/" + id
}

func mapZhihuColumnItem(entry zhihuColumnItem) *models.Item {
	switch entry.Type {
	case "article":
		link := entry.URL
		if link == "" {
			link = fmt.Sprintf("https://zhuanlan.zhihu.com/p/%d", entry.ID)
		}
		item := routeutils.NewItem(entry.Title, link, processZhihuContent(entry.Content), time.Unix(entry.Created, 0))
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-article-%d", entry.ID)
			routeutils.SetItemAuthor(item, entry.Author.Name, "", "")
		}
		return item
	case "answer":
		link := fmt.Sprintf("https://www.zhihu.com/question/%d/answer/%d", entry.Question.ID, entry.ID)
		desc := processZhihuContent(entry.Content)
		if desc == "" {
			desc = "<p>" + entry.Excerpt + "</p>"
		}
		item := routeutils.NewItem(entry.Question.Title, link, desc, time.Unix(entry.CreatedTime, 0))
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-answer-%d", entry.ID)
			routeutils.SetItemAuthor(item, entry.Author.Name, "", "")
		}
		return item
	case "zvideo":
		link := fmt.Sprintf("https://www.zhihu.com/zvideo/%d", entry.ID)
		desc := entry.Description
		if desc == "" {
			desc = "视频内容请跳转至原页面观看"
		} else {
			desc += "<br/><br/>视频内容请跳转至原页面观看"
		}
		item := routeutils.NewItem(entry.Title, link, "<p>"+desc+"</p>", time.Unix(entry.CreatedAt, 0))
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-zvideo-%d", entry.ID)
			routeutils.SetItemAuthor(item, entry.Author.Name, "", "")
		}
		return item
	default:
		return nil
	}
}

// ---------------- 知乎问题 /zhihu/question/:questionId ----------------

type zhihuAnswer struct {
	ID          zhihuInt64 `json:"id"`
	Content     string     `json:"content"`
	Excerpt     string     `json:"excerpt"`
	VoteupCount int        `json:"voteup_count"`
	CreatedTime int64      `json:"created_time"`
	UpdatedTime int64      `json:"updated_time"`
	Author      struct {
		Name string `json:"name"`
	} `json:"author"`
	Question struct {
		ID    zhihuInt64 `json:"id"`
		Title string     `json:"title"`
	} `json:"question"`
}

type zhihuAnswersResp struct {
	Data []zhihuAnswer `json:"data"`
}

// include 字段与上游一致：内容 + 元数据 + 所属问题标题
const zhihuAnswerInclude = "data[*].content,excerpt,voteup_count,created_time,updated_time,author.name;data[*].question.title,id"

var zhihuQuestionRoute = routeutils.RouteSpec{
	Path:        "question/:questionId",
	Name:        "知乎问题",
	Example:     "zhihu/question/59895982",
	Maintainers: []string{"xihale"},
	Description: "知乎问题的全部回答",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true, EnvDeps: []string{zhihuCookiesEnv}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("questionId", "问题 id"),
		routeutils.OptionalParam("sort_by", "排序方式：default（默认）、created、updated"),
		routeutils.OptionalParam("limit", "条数，默认 20，上限 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  ZhihuQuestionHandler,
}

// ZhihuQuestionHandler handles /zhihu/question/:questionId
func ZhihuQuestionHandler(c *ctxpkg.Context) (*models.Feed, error) {
	questionID := c.Param("questionId")
	sortBy := routeutils.ParseEnum(c.QueryParam("sort_by"), "default", "default", "created", "updated")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()
	if err := requireZhihuCookies(); err != nil {
		return nil, err
	}

	pageURL := questionLink(parseZhihuID(questionID))
	q := url.Values{}
	q.Set("include", zhihuAnswerInclude)
	q.Set("offset", "0")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("sort_by", sortBy)
	q.Set("platform", "desktop")
	url := zhihuAPIBase + "/questions/" + questionID + "/answers?" + q.Encode()
	var resp zhihuAnswersResp
	if err := zhihuProfile(pageURL).Fetch(url).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("该问题暂无回答或不存在")
	}

	qTitle := resp.Data[0].Question.Title
	feedTitle := "知乎-" + qTitle
	feed := routeutils.NewFeed(feedTitle, pageURL, qTitle)
	routeutils.AppendMappedItems(feed, resp.Data, 0, func(a zhihuAnswer) *models.Item {
		link := fmt.Sprintf("https://www.zhihu.com/question/%s/answer/%d", questionID, a.ID)
		desc := processZhihuContent(a.Content)
		if desc == "" && a.Excerpt != "" {
			desc = "<p>" + a.Excerpt + "</p>"
		}
		meta := fmt.Sprintf("<p>赞同：%d</p>", a.VoteupCount)
		item := routeutils.NewItem(answerTitle(a.Author.Name, a.Excerpt), link, desc+meta, time.Unix(a.UpdatedTimeOrCreated(), 0))
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-answer-%d", a.ID)
			routeutils.SetItemAuthor(item, a.Author.Name, "", "")
		}
		return item
	})
	return feed, nil
}

func (a zhihuAnswer) UpdatedTimeOrCreated() int64 {
	if a.UpdatedTime > 0 {
		return a.UpdatedTime
	}
	return a.CreatedTime
}

// answerTitle builds "{作者}的回答：{摘要}"，摘要截断避免超长标题。
func answerTitle(author, excerpt string) string {
	t := author + " 的回答"
	if excerpt == "" {
		return t
	}
	runes := []rune(excerpt)
	if len(runes) > 60 {
		excerpt = string(runes[:60]) + "…"
	}
	return t + "：" + excerpt
}

func parseZhihuID(raw string) zhihuInt64 {
	id, _ := strconv.ParseInt(raw, 10, 64)
	return zhihuInt64(id)
}
