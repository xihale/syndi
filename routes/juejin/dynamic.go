package routes

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// juejinDynamicTargetData is a polymorphic entry body: only the subset of
// fields matching TargetType is populated.
type juejinDynamicTargetData struct {
	// short_msg (沸点/点赞沸点)
	MsgID          string           `json:"msg_id"`
	MsgInfo        juejinPinMsgInfo `json:"msg_Info"`
	AuthorUserInfo juejinAuthorInfo `json:"author_user_info"`
	Topic          struct {
		Title string `json:"title"`
	} `json:"topic"`

	// article
	ArticleID   string                `json:"article_id"`
	ArticleInfo juejinArticleInfo     `json:"article_info"`
	Category    juejinArticleCategory `json:"category"`
	Tags        []juejinTag           `json:"tags"`

	// user follow / tag follow
	UserName    string `json:"user_name"`
	UserID      string `json:"user_id"`
	Description string `json:"description"`
	TagName     string `json:"tag_name"`
}

type juejinDynamicEntry struct {
	Action     int                     `json:"action"` // 0 发文章 1 赞文章 2 发沸点 3 赞沸点 4 关注用户 5 关注标签
	Time       int64                   `json:"time"`   // unix seconds
	TargetType string                  `json:"target_type"`
	User       juejinAuthorInfo        `json:"user"`
	TargetData juejinDynamicTargetData `json:"target_data"`
}

type juejinDynamicResp struct {
	Data struct {
		List []juejinDynamicEntry `json:"list"`
	} `json:"data"`
}

var juejinDynamicRoute = routeutils.RouteSpec{
	Path:        "dynamic/:id",
	Name:        "Juejin User Dynamic",
	Example:     "juejin/dynamic/3051900006845944",
	Maintainers: []string{"xihale"},
	Description: "掘金用户动态（文章、沸点与关注行为）",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "用户 id，可在用户页 URL 中找到"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  JuejinDynamicHandler,
}

// mapJuejinDynamicEntries converts user dynamic entries into feed items.
func mapJuejinDynamicEntries(feed *models.Feed, owner string, entries []juejinDynamicEntry) {
	for _, e := range entries {
		td := e.TargetData
		switch e.TargetType {
		case "short_msg":
			content := strings.TrimSpace(td.MsgInfo.Content)
			if content == "" || td.MsgID == "" {
				continue
			}
			var b strings.Builder
			if e.Action == 3 { // 赞了沸点
				title := fmt.Sprintf("%s 赞了这篇沸点//@%s：%s", owner, td.AuthorUserInfo.UserName, content)
				b.WriteString("<p>" + html.EscapeString(title) + "</p>")
			} else {
				b.WriteString("<p>" + html.EscapeString(content) + "</p>")
			}
			for _, img := range td.MsgInfo.PicList {
				b.WriteString(`<img src="` + html.EscapeString(img) + `"/><br>`)
			}
			pubDate := parseJuejinUnixSeconds(td.MsgInfo.Ctime)
			item := routeutils.NewItem(content, "https://juejin.cn/pin/"+td.MsgID, b.String(), pubDate)
			if e.Action == 3 {
				item.Title = fmt.Sprintf("%s 赞了这篇沸点//@%s：%s", owner, td.AuthorUserInfo.UserName, content)
			}
			routeutils.SetItemAuthor(item, td.AuthorUserInfo.UserName, "", "")
			if td.Topic.Title != "" {
				routeutils.SetCategories(item, td.Topic.Title)
			}
			routeutils.AddItem(feed, item)
		case "article":
			info := td.ArticleInfo
			title := strings.TrimSpace(info.Title)
			if title == "" || info.ArticleID == "" {
				continue
			}
			if e.Action == 1 { // 赞了文章
				title = fmt.Sprintf("%s 赞了这篇文章//@%s：%s", owner, td.AuthorUserInfo.UserName, title)
			}
			desc := ""
			if brief := strings.TrimSpace(info.BriefContent); brief != "" {
				desc = "<p>" + html.EscapeString(brief) + "</p>"
			}
			link := "https://juejin.cn/post/" + info.ArticleID
			item := routeutils.NewItem(title, link, desc, parseJuejinUnixSeconds(info.Ctime))
			routeutils.SetItemAuthor(item, td.AuthorUserInfo.UserName, "", "")
			cats := make([]string, 0, len(td.Tags)+1)
			if td.Category.CategoryName != "" {
				cats = append(cats, td.Category.CategoryName)
			}
			for _, t := range td.Tags {
				if t.TagName != "" {
					cats = append(cats, t.TagName)
				}
			}
			routeutils.SetCategories(item, cats...)
			routeutils.AddItem(feed, item)
		case "user":
			if td.UserName == "" || td.UserID == "" {
				continue
			}
			title := fmt.Sprintf("%s 关注了 %s", owner, td.UserName)
			desc := ""
			if bio := strings.TrimSpace(td.Description); bio != "" {
				desc = "<p>简介：" + html.EscapeString(bio) + "</p>"
			}
			pubDate := time.Time{}
			if e.Time > 0 {
				pubDate = time.Unix(e.Time, 0)
			}
			item := routeutils.NewItem(title, "https://juejin.cn/user/"+td.UserID, desc, pubDate)
			routeutils.SetItemAuthor(item, td.UserName, "", "")
			routeutils.AddItem(feed, item)
		case "tag":
			if td.TagName == "" {
				continue
			}
			title := fmt.Sprintf("%s 关注了标签 %s", owner, td.TagName)
			link := "https://juejin.cn/tag/" + url.PathEscape(td.TagName)
			pubDate := time.Time{}
			if e.Time > 0 {
				pubDate = time.Unix(e.Time, 0)
			}
			item := routeutils.NewItem(title, link, "<p>"+html.EscapeString(td.TagName)+"</p>", pubDate)
			routeutils.SetCategories(item, td.TagName)
			routeutils.AddItem(feed, item)
		default:
			// Unknown activity types are skipped rather than failing the feed.
			continue
		}
	}
}

// parseJuejinUnixSeconds parses unix-second JSON strings ("1739...").
func parseJuejinUnixSeconds(raw string) time.Time {
	t := time.Time{}
	if sec, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil && sec > 0 {
		t = time.Unix(sec, 0)
	}
	return t
}

// JuejinDynamicHandler handles /juejin/dynamic/:id
func JuejinDynamicHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")

	var resp juejinDynamicResp
	endpoint := fmt.Sprintf("%s/user_api/v1/user/dynamic?user_id=%s&cursor=0", juejinAPIBaseURL, id)
	if err := juejinProfile.Fetch(endpoint).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if len(resp.Data.List) == 0 {
		return nil, fmt.Errorf("juejin: no dynamics found for user %q", id)
	}

	user := resp.Data.List[0].User
	name := user.UserName
	if name == "" {
		name = id
	}
	description := user.Description
	if description == "" {
		description = "掘金用户动态-" + name
	}
	feed := routeutils.NewFeedWithOptions(routeutils.FeedOptions{
		Title:       "掘金用户动态-" + name,
		Link:        "https://juejin.cn/user/" + id + "/",
		Description: description,
		Image:       user.AvatarLarge,
	})
	mapJuejinDynamicEntries(feed, name, resp.Data.List)
	return feed, nil
}
