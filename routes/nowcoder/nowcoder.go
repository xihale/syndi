package routes

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const nowCoderBaseURL = "https://www.nowcoder.com"

func nowCoderProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer(nowCoderBaseURL + "/")
}

// Routes lists all NowCoder route specs in this package.

var nowCoderHotsRoute = routeutils.RouteSpec{
	Path:        "hots",
	Name:        "NowCoder Hot Discussions",
	Example:     "nowcoder/hots",
	Maintainers: []string{"xihale"},
	Description: "NowCoder (nowcoder.com) trending discussion subjects, same as type=1",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     NowCoderHotsHandler,
}

var nowCoderHotsTypeRoute = routeutils.RouteSpec{
	Path:        "hots/:type",
	Name:        "NowCoder Hot List By Type",
	Example:     "nowcoder/hots/1",
	Maintainers: []string{"xihale"},
	Description: "NowCoder hot list. Type 1 = trending discussion subjects, type 2 = site-wide hot posts. Optional query limit (default 20, max 50)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("type", "Hot list type: 1 trending discussion subjects (default), 2 site-wide hot posts"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  NowCoderHotsHandler,
}

var nowCoderScheduleRoute = routeutils.RouteSpec{
	Path:        "schedule",
	Name:        "NowCoder Campus Recruiting Schedule",
	Example:     "nowcoder/schedule",
	Maintainers: []string{"xihale"},
	Description: "Campus recruiting schedule of famous companies on NowCoder, same as propertyId=0&typeId=0",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    6 * time.Hour,
	Handler:     NowCoderScheduleHandler,
}

var nowCoderSchedulePropertyRoute = routeutils.RouteSpec{
	Path:        "schedule/:propertyId",
	Name:        "NowCoder Campus Schedule By Industry",
	Example:     "nowcoder/schedule/1",
	Maintainers: []string{"xihale"},
	Description: "Campus recruiting schedule filtered by industry id, same as typeId=0",
	Categories:  []models.Category{{Name: "programming"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("propertyId", "Industry id from the schedule page API, 0 for all"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  NowCoderScheduleHandler,
}

var nowCoderSchedulePropertyTypeRoute = routeutils.RouteSpec{
	Path:        "schedule/:propertyId/:typeId",
	Name:        "NowCoder Campus Schedule By Industry And Category",
	Example:     "nowcoder/schedule/1/2",
	Maintainers: []string{"xihale"},
	Description: "Campus recruiting schedule filtered by industry id and job category id",
	Categories:  []models.Category{{Name: "programming"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("propertyId", "Industry id from the schedule page API, 0 for all"),
		routeutils.RequiredParam("typeId", "Job category id from the schedule page API, 0 for all"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  NowCoderScheduleHandler,
}

var nowCoderInterviewRoute = routeutils.RouteSpec{
	Path:        "interview/:jobId",
	Name:        "NowCoder Interview Experiences",
	Example:     "nowcoder/interview/11200",
	Maintainers: []string{"xihale"},
	Description: "Latest interview experience posts on NowCoder for a job position. jobId 11200 = all positions, 11002 = Java, etc.",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("jobId", "Job position id, e.g. 11200 for all, 11002 for Java"),
		routeutils.OptionalParam("limit", "Max items, default 20, max 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  NowCoderInterviewHandler,
}

// NowCoderHotsHandler handles /nowcoder/hots and /nowcoder/hots/:type.
func NowCoderHotsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	hotType := c.Param("type")
	if hotType == "" {
		hotType = "1"
	}
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()

	switch hotType {
	case "1":
		apiURL := fmt.Sprintf("https://gw-c.nowcoder.com/api/sparta/subject/hot-subject?limit=%d", limit)
		var resp ncHotSubjectResp
		if err := nowCoderProfile().JSONAccept().Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
			return nil, err
		}
		if resp.Code != 0 {
			return nil, fmt.Errorf("nowcoder hot-subject api error code %d: %s", resp.Code, resp.Msg)
		}
		feed := routeutils.NewFeed(
			"牛客网-热议话题",
			"https://mnowpick.nowcoder.com/m/discuss/hot",
			"牛客网热议话题热榜",
		)
		routeutils.AppendMappedItems(feed, resp.Data.Result, 0, func(r ncHotSubject) *models.Item {
			if r.UUID == "" || r.Content == "" {
				return nil
			}
			link := fmt.Sprintf("%s/creation/subject/%s", nowCoderBaseURL, r.UUID)
			desc := fmt.Sprintf(`<img src="%s" alt="rank"/><br/>浏览 %d · 帖子 %d`,
				html.EscapeString(r.NumberIcon), r.ViewCount.Int64(), r.MomentCount.Int64())
			item := routeutils.NewItem(r.Content, link, desc, time.Time{})
			item.GUID = "nowcoder-subject-" + r.UUID
			return item
		})
		return feed, nil

	case "2":
		apiURL := fmt.Sprintf("https://gw-c.nowcoder.com/api/sparta/hot-search/top-hot-pc?size=%d", limit)
		var resp ncTopHotResp
		if err := nowCoderProfile().JSONAccept().Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
			return nil, err
		}
		if resp.Code != 0 {
			return nil, fmt.Errorf("nowcoder top-hot-pc api error code %d: %s", resp.Code, resp.Msg)
		}
		feed := routeutils.NewFeed(
			"牛客网-全站热贴",
			"https://mnowpick.nowcoder.com/m/discuss/hot",
			"牛客网全站热贴榜",
		)
		routeutils.AppendMappedItems(feed, resp.Data.Result, 0, func(r ncTopHotPost) *models.Item {
			if r.Title == "" || r.UUID == "" {
				return nil
			}
			link := fmt.Sprintf("%s/feed/main/detail/%s", nowCoderBaseURL, r.UUID)
			item := routeutils.NewItem(r.Title, link, "", time.Time{})
			item.GUID = "nowcoder-hotpost-" + r.UUID
			return item
		})
		return feed, nil

	default:
		return nil, fmt.Errorf("invalid type parameter %q: must be 1 or 2", hotType)
	}
}

// NowCoderScheduleHandler handles /nowcoder/schedule[/propertyId[/typeId]].
func NowCoderScheduleHandler(c *ctxpkg.Context) (*models.Feed, error) {
	propertyID := c.Param("propertyId")
	if propertyID == "" {
		propertyID = "0"
	}
	typeID := c.Param("typeId")
	if typeID == "" {
		typeID = "0"
	}
	ctx := c.Parent()

	apiURL := fmt.Sprintf("%s/school/schedule/data?token=&query=&typeId=%s&propertyId=%s&onlyFollow=false",
		nowCoderBaseURL, url.QueryEscape(typeID), url.QueryEscape(propertyID))
	var resp ncScheduleResp
	if err := nowCoderProfile().JSONAccept().Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("nowcoder schedule api error code %d: %s", resp.Code, resp.Msg)
	}

	feed := routeutils.NewFeed(
		"名企校招日程",
		nowCoderBaseURL+"/school/schedule",
		"牛客名企校招日程",
	)
	routeutils.AppendMappedItems(feed, resp.Data.CompanyList, 0, func(co ncScheduleCompany) *models.Item {
		if co.Name == "" || co.ID.String() == "" {
			return nil
		}
		link := fmt.Sprintf("%s/school/schedule/%s", nowCoderBaseURL, co.ID.String())

		var sb strings.Builder
		sb.WriteString("<table>")
		logo := firstNonEmptyStr(co.LogoRadius, co.Logo, co.HomeLogo)
		if logo != "" {
			fmt.Fprintf(&sb, `<tr><td><img src="%s" referrerpolicy="no-referrer"/></td></tr>`, html.EscapeString(logo))
		}
		for _, sc := range co.Schedules {
			content := html.EscapeString(sc.Content)
			if content == "" {
				content = html.EscapeString(sc.Name)
			}
			fmt.Fprintf(&sb, "<tr><td>%s</td><td>%s</td></tr>", content, html.EscapeString(sc.Time))
		}
		if len(co.Cities) > 0 {
			cities := make([]string, 0, len(co.Cities))
			for _, city := range co.Cities {
				cities = append(cities, html.EscapeString(city))
			}
			fmt.Fprintf(&sb, "<tr><td>城市</td><td>%s</td></tr>", strings.Join(cities, "、"))
		}
		sb.WriteString("</table>")

		var pubDate time.Time
		if ms := co.CreateTime.Int64(); ms > 0 {
			pubDate = time.UnixMilli(ms)
		}
		item := routeutils.NewItem(co.Name, link, sb.String(), pubDate)
		item.GUID = "nowcoder-schedule-" + co.ID.String()
		return item
	})
	return feed, nil
}

// NowCoderInterviewHandler handles /nowcoder/interview/:jobId.
func NowCoderInterviewHandler(c *ctxpkg.Context) (*models.Feed, error) {
	jobID := c.Param("jobId")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()

	payload := map[string]any{"jobId": jobID, "level": 3, "order": 3, "page": 1}
	var resp ncInterviewResp
	if err := nowCoderProfile().JSONAccept().
		PostJSON("https://gw-c.nowcoder.com/api/sparta/job-experience/experience/job/list", payload).
		GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("nowcoder interview api error code %d: %s", resp.Code, resp.Msg)
	}

	feed := routeutils.NewFeed(
		"牛客网-面试经验",
		nowCoderBaseURL+"/interview/",
		"牛客网面试经验",
	)
	count := 0
	for _, rec := range resp.Data.Records {
		if count >= limit {
			break
		}
		post := rec.ContentData
		if post.UUID == "" {
			post = rec.MomentData
		}
		if post.UUID == "" || (post.Title == "" && post.Content == "") {
			continue
		}
		title := post.Title
		if title == "" {
			title = truncateNowCoderText(post.Content, 60)
		}
		link := fmt.Sprintf("%s/feed/main/detail/%s", nowCoderBaseURL, post.UUID)

		body := strings.TrimSpace(post.Content)
		desc := "<p>" + strings.ReplaceAll(html.EscapeString(body), "\n", "<br/>") + "</p>"
		if author := rec.UserBrief.Nickname; author != "" {
			desc += fmt.Sprintf(`<br/>作者：%s`, html.EscapeString(author))
		}

		var pubDate time.Time
		if ms := post.CreateTime.Int64(); ms > 0 {
			pubDate = time.UnixMilli(ms)
		}
		item := routeutils.NewItem(title, link, desc, pubDate)
		item.GUID = "nowcoder-interview-" + post.UUID
		if author := rec.UserBrief.Nickname; author != "" {
			routeutils.SetAuthor(item, author)
		}
		routeutils.AddItem(feed, item)
		count++
	}
	return feed, nil
}

// firstNonEmptyStr returns the first non-empty string.
func firstNonEmptyStr(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// truncateNowCoderText shortens plain text for use as a title.
func truncateNowCoderText(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

type ncEnvelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ncHotSubjectResp is the payload of /api/sparta/subject/hot-subject.
type ncHotSubjectResp struct {
	ncEnvelope
	Data struct {
		Result []ncHotSubject `json:"result"`
	} `json:"data"`
}

type ncHotSubject struct {
	UUID        string    `json:"uuid"`
	Content     string    `json:"content"`
	NumberIcon  string    `json:"numberIcon"`
	ViewCount   flexInt64 `json:"viewCount"`
	MomentCount flexInt64 `json:"momentCount"`
	HotValue    flexInt64 `json:"hotValue"`
}

// ncTopHotResp is the payload of /api/sparta/hot-search/top-hot-pc.
type ncTopHotResp struct {
	ncEnvelope
	Data struct {
		Result []ncTopHotPost `json:"result"`
	} `json:"data"`
}

type ncTopHotPost struct {
	Title string     `json:"title"`
	UUID  string     `json:"uuid"`
	ID    flexString `json:"id"`
}

// ncScheduleResp is the payload of /school/schedule/data.
type ncScheduleResp struct {
	ncEnvelope
	Data struct {
		CompanyList []ncScheduleCompany `json:"companyList"`
	} `json:"data"`
}

type ncScheduleCompany struct {
	ID         flexString `json:"id"`
	Name       string     `json:"name"`
	Logo       string     `json:"logo"`
	LogoRadius string     `json:"logoRadius"`
	HomeLogo   string     `json:"homeLogo"`
	CreateTime flexInt64  `json:"createTime"`
	Cities     []string   `json:"cities"`
	Schedules  []struct {
		Name    string `json:"name"`
		Content string `json:"content"`
		Time    string `json:"time"`
	} `json:"schedules"`
}

// ncInterviewResp is the payload of the job-experience POST API.
type ncInterviewResp struct {
	ncEnvelope
	Data struct {
		Records []ncInterviewRecord `json:"records"`
	} `json:"data"`
}

type ncInterviewRecord struct {
	ContentData ncInterviewPost `json:"contentData"`
	MomentData  ncInterviewPost `json:"momentData"`
	UserBrief   struct {
		Nickname string `json:"nickname"`
	} `json:"userBrief"`
}

type ncInterviewPost struct {
	UUID       string    `json:"uuid"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	CreateTime flexInt64 `json:"createTime"`
}
