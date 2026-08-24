package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

// ---------------- 用户回答 /zhihu/people/answers/:id ----------------

var zhihuPeopleAnswersRoute = routeutils.RouteSpec{
	Path:        "people/answers/:id",
	Name:        "知乎用户回答",
	Example:     "zhihu/people/answers/diygod",
	Maintainers: []string{"xihale"},
	Description: "指定用户的最新回答",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "用户 url token，可在用户主页 URL 中找到"),
		routeutils.OptionalParam("limit", "条数，默认 20，上限 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  ZhihuPeopleAnswersHandler,
}

// ZhihuPeopleAnswersHandler handles /zhihu/people/answers/:id
func ZhihuPeopleAnswersHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()
	if err := requireZhihuCookies(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("include", zhihuAnswerInclude)
	q.Set("offset", "0")
	q.Set("limit", strconv.Itoa(limit))
	apiURL := fmt.Sprintf("%s/members/%s/answers?%s", zhihuAPIBase, id, q.Encode())
	var resp zhihuAnswersResp
	if err := zhihuProfile("https://www.zhihu.com/people/"+id).Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}

	feedTitle := fmt.Sprintf("%s 的知乎回答", id)
	if len(resp.Data) > 0 && resp.Data[0].Author.Name != "" {
		feedTitle = fmt.Sprintf("%s 的知乎回答", resp.Data[0].Author.Name)
	}
	feed := routeutils.NewFeed(feedTitle, "https://www.zhihu.com/people/"+id+"/answers", "")
	routeutils.AppendMappedItems(feed, resp.Data, 0, func(a zhihuAnswer) *models.Item {
		link := fmt.Sprintf("https://www.zhihu.com/question/%d/answer/%d", a.Question.ID, a.ID)
		desc := processZhihuContent(a.Content)
		if desc == "" && a.Excerpt != "" {
			desc = "<p>" + html.EscapeString(a.Excerpt) + "</p>"
		}
		desc += fmt.Sprintf("<p>赞同：%d</p>", a.VoteupCount)
		item := routeutils.NewItem(a.Question.Title, link, desc, time.Unix(a.UpdatedTimeOrCreated(), 0))
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-answer-%d", a.ID)
			routeutils.SetItemAuthor(item, a.Author.Name, "", "")
		}
		return item
	})
	return feed, nil
}

// ---------------- 用户文章 /zhihu/posts/:usertype/:id ----------------

type zhihuMemberProfile struct {
	Name      string `json:"name"`
	Headline  string `json:"headline"`
	AvatarURL string `json:"avatar_url"`
}

type zhihuArticle struct {
	ID      zhihuInt64 `json:"id"`
	Title   string     `json:"title"`
	Content string     `json:"content"`
	URL     string     `json:"url"`
	Created int64      `json:"created"`
	Updated int64      `json:"updated"`
	Author  struct {
		Name string `json:"name"`
	} `json:"author"`
}

type zhihuArticlesResp struct {
	Data []zhihuArticle `json:"data"`
}

func (a zhihuArticle) articleLink() string {
	if a.URL != "" {
		return a.URL
	}
	return fmt.Sprintf("https://zhuanlan.zhihu.com/p/%d", a.ID)
}

var zhihuPostsRoute = routeutils.RouteSpec{
	Path:        "posts/:usertype/:id",
	Name:        "知乎用户文章",
	Example:     "zhihu/posts/people/frederchen",
	Maintainers: []string{"xihale"},
	Description: "用户发布的专栏文章",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("usertype", "用户类型：people（普通用户）或 org（机构账号）"),
		routeutils.RequiredParam("id", "用户 url token"),
		routeutils.OptionalParam("limit", "条数，默认 20，上限 50"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  ZhihuPostsHandler,
}

// ZhihuPostsHandler handles /zhihu/posts/:usertype/:id
func ZhihuPostsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	usertype := routeutils.ParseEnum(c.Param("usertype"), "people", "people", "org")
	id := c.Param("id")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()
	if err := requireZhihuCookies(); err != nil {
		return nil, err
	}

	homeURL := fmt.Sprintf("https://www.zhihu.com/%s/%s/", usertype, id)

	var profile zhihuMemberProfile
	if err := zhihuProfile(homeURL).Fetch(zhihuAPIBase+"/members/"+id).GetJSON(ctx, c.Client(), &profile); err != nil {
		return nil, fmt.Errorf("获取用户信息失败（请确认用户存在且 %s 有效）: %w", zhihuCookiesEnv, err)
	}

	q := url.Values{}
	q.Set("include", "data[*].content,title,created,updated,url")
	q.Set("offset", "0")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("sort_by", "created")
	apiURL := fmt.Sprintf("%s/members/%s/articles?%s", zhihuAPIBase, id, q.Encode())
	var resp zhihuArticlesResp
	if err := zhihuProfile(homeURL+"posts").Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}

	name := profile.Name
	if name == "" {
		name = id
	}
	feedTitle := name + " 的知乎文章"
	feed := routeutils.NewFeed(feedTitle, strings.TrimSuffix(homeURL, "/")+"/posts", profile.Headline)
	routeutils.AppendMappedItems(feed, resp.Data, 0, func(a zhihuArticle) *models.Item {
		item := routeutils.NewItem(a.Title, a.articleLink(), processZhihuContent(a.Content), time.Unix(a.Created, 0))
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-article-%d", a.ID)
			routeutils.SetItemAuthor(item, a.Author.Name, "", "")
		}
		return item
	})
	return feed, nil
}

// ---------------- 用户动态 /zhihu/people/activities/:id ----------------

// zhihuFlexContent handles the polymorphic upstream "content" field: an HTML
// string (answer/article) or an array of rich-text parts (pin).
type zhihuFlexContent struct {
	HTML  string
	Parts []struct {
		Type    string `json:"type"`
		OwnText string `json:"own_text"`
	}
}

func (f *zhihuFlexContent) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		f.HTML = s
		return nil
	}
	var parts []struct {
		Type    string `json:"type"`
		OwnText string `json:"own_text"`
	}
	if err := json.Unmarshal(data, &parts); err == nil {
		f.Parts = parts
		return nil
	}
	return fmt.Errorf("zhihu: unexpected content payload")
}

func (f zhihuFlexContent) String() string { return f.HTML }

type zhihuMomentTargetData struct {
	Type         string           `json:"type"`
	ID           zhihuInt64       `json:"id"`
	Title        string           `json:"title"`
	Name         string           `json:"name"` // topic/column/roundtable
	Subject      string           `json:"subject"`
	URL          string           `json:"url"`
	Content      zhihuFlexContent `json:"content"`
	Excerpt      string           `json:"excerpt"`
	Intro        string           `json:"intro"`
	ExcerptTitle string           `json:"excerpt_title"`
	Detail       string           `json:"detail"`
	Created      int64            `json:"created"`
	CreatedTime  int64            `json:"created_time"`
	CreatedAt    int64            `json:"created_at"`
	UpdatedTime  int64            `json:"updated_time"`
	Author       struct {
		Name string `json:"name"`
	} `json:"author"`
	Question struct {
		ID    zhihuInt64 `json:"id"`
		Title string     `json:"title"`
	} `json:"question"`
}

type zhihuMoment struct {
	Actor struct {
		Name      string `json:"name"`
		Headline  string `json:"headline"`
		AvatarURL string `json:"avatar_url"`
	} `json:"actor"`
	ActionText  string                `json:"action_text"`
	CreatedTime int64                 `json:"created_time"`
	Target      zhihuMomentTargetData `json:"target"`
}

type zhihuMomentsResp struct {
	Data []zhihuMoment `json:"data"`
}

var zhihuActivitiesRoute = routeutils.RouteSpec{
	Path:        "people/activities/:id",
	Name:        "知乎用户动态",
	Example:     "zhihu/people/activities/diygod",
	Maintainers: []string{"xihale"},
	Description: "用户的回答、文章、想法等公开动态",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "用户 url token"),
		routeutils.OptionalParam("limit", "条数，默认 20，上限 20"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  ZhihuActivitiesHandler,
}

// ZhihuActivitiesHandler handles /zhihu/people/activities/:id
func ZhihuActivitiesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 20)
	ctx := c.Parent()
	if err := requireZhihuCookies(); err != nil {
		return nil, err
	}

	pageURL := "https://www.zhihu.com/people/" + id
	mq := url.Values{}
	mq.Set("limit", strconv.Itoa(limit))
	mq.Set("desktop", "true")
	mq.Set("ws_qiangzhisafe", "0")
	apiURL := fmt.Sprintf("https://www.zhihu.com/api/v3/moments/%s/activities?%s", id, mq.Encode())
	var resp zhihuMomentsResp
	if err := zhihuProfile(pageURL).Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("未获取到该用户的公开动态（账号可能无公开活动或 %s 无权限查看）", zhihuCookiesEnv)
	}

	actor := resp.Data[0].Actor
	actorName := actor.Name
	if actorName == "" {
		actorName = id
	}
	feed := routeutils.NewFeed(actorName+" 的知乎动态", pageURL+"/activities", actor.Headline)

	routeutils.AppendMappedItems(feed, resp.Data, 0, func(m zhihuMoment) *models.Item {
		title, link, desc, author, pub := zhihuMomentTarget(m.Target)
		if title == "" || link == "" {
			return nil
		}
		fullTitle := actorName + m.ActionText + ": " + title
		item := routeutils.NewItem(fullTitle, link, desc, pub)
		if item != nil {
			item.GUID = fmt.Sprintf("zhihu-moment-%s-%d", m.Target.Type, m.Target.ID)
			if author != "" {
				routeutils.SetItemAuthor(item, author, "", "")
			}
		}
		return item
	})
	return feed, nil
}

// zhihuMomentTarget maps an activity target to feed fields.
func zhihuMomentTarget(t zhihuMomentTargetData) (title, link, desc, author string, pub time.Time) {
	switch t.Type {
	case "answer":
		title = t.Question.Title
		link = fmt.Sprintf("https://www.zhihu.com/question/%d/answer/%d", t.Question.ID, t.ID)
		desc = processZhihuContent(t.Content.String())
		if desc == "" && t.Excerpt != "" {
			desc = "<p>" + html.EscapeString(t.Excerpt) + "</p>"
		}
		author = t.Author.Name
		pub = time.Unix(firstNonZero(t.UpdatedTime, t.CreatedTime), 0)
	case "article":
		title = t.Title
		link = t.URL
		if link == "" {
			link = fmt.Sprintf("https://zhuanlan.zhihu.com/p/%d", t.ID)
		}
		desc = processZhihuContent(t.Content.String())
		author = t.Author.Name
		pub = time.Unix(firstNonZero(t.Created, t.CreatedTime), 0)
	case "pin":
		title = t.ExcerptTitle
		link = fmt.Sprintf("https://www.zhihu.com/pin/%d", t.ID)
		var parts []string
		for _, pc := range t.Content.Parts {
			switch pc.Type {
			case "text":
				parts = append(parts, "<p>"+html.EscapeString(pc.OwnText)+"</p>")
			case "image":
				// 图片 URL 在 content[].image 数组中，此处仅保留文字摘要
			default:
				// link/link_card/video 跳转原页即可
			}
		}
		if len(parts) == 0 && t.Excerpt != "" {
			parts = append(parts, "<p>"+html.EscapeString(t.Excerpt)+"</p>")
		}
		desc = strings.Join(parts, "")
		author = t.Author.Name
		pub = time.Unix(firstNonZero(t.UpdatedTime, t.CreatedTime), 0)
	case "question":
		title = t.Title
		link = questionLink(t.ID)
		desc = processZhihuContent(t.Detail)
		pub = time.Unix(firstNonZero(t.Created, t.CreatedTime), 0)
	case "column":
		title = t.Title
		link = fmt.Sprintf("https://zhuanlan.zhihu.com/%d", t.ID)
		if t.Intro != "" {
			desc = "<p>" + html.EscapeString(t.Intro) + "</p>"
		}
	case "topic":
		title = t.Name
		link = fmt.Sprintf("https://www.zhihu.com/topic/%d", t.ID)
	case "live":
		title = t.Subject
		link = fmt.Sprintf("https://www.zhihu.com/lives/%d", t.ID)
	case "roundtable":
		title = t.Name
		link = fmt.Sprintf("https://www.zhihu.com/roundtable/%d", t.ID)
	default:
		// collection 等少见类型的链接结构不稳定，跳过避免产出坏条目
		return "", "", "", "", time.Time{}
	}
	return title, link, desc, author, pub
}

func firstNonZero(values ...int64) int64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}
