package routes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// weiboUserRoute serves /weibo/user/:uid — a user's latest posts.
var weiboUserRoute = routeutils.RouteSpec{
	Path:        "user/:uid",
	Name:        "User Timeline",
	Example:     "weibo/user/1195230310",
	Maintainers: []string{"xihale"},
	Description: "Latest posts of a Weibo user by numeric uid",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true, EnvDeps: []string{weiboCookiesEnv}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("uid", "Numeric weibo user id, from the profile URL"),
		routeutils.OptionalParam("showRetweeted", "Set 0 to hide reposts, default keeps them"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  WeiboUserHandler,
}

type weiboUserInfo struct {
	ScreenName string `json:"screen_name"`
	Desc       string `json:"description"`
}

type weiboTabsInfo struct {
	Tabs []struct {
		TabType     string `json:"tab_type"`
		ContainerID string `json:"containerid"`
	} `json:"tabs"`
}

type weiboMBlog struct {
	ID             string           `json:"id"`
	Bid            string           `json:"bid"`
	CreatedAt      string           `json:"created_at"`
	Text           string           `json:"text"`
	RepostsCount   int              `json:"reposts_count"`
	CommentsCount  int              `json:"comments_count"`
	AttitudesCount int              `json:"attitudes_count"`
	Retweeted      *weiboRetweetRef `json:"retweeted_status"`
}

type weiboRetweetRef struct {
	Bid       string `json:"bid"`
	CreatedAt string `json:"created_at"`
	Text      string `json:"text"`
}

type weiboCard struct {
	MBlog *weiboMBlog `json:"mblog"`
}

type weiboContainerResp struct {
	OK   int `json:"ok"`
	Data struct {
		UserInfo weiboUserInfo `json:"userInfo"`
		TabsInfo weiboTabsInfo `json:"tabsInfo"`
	} `json:"data"`
}

type weiboCardsResp struct {
	OK   int `json:"ok"`
	Data struct {
		Cards []weiboCard `json:"cards"`
	} `json:"data"`
}

// weiboTagRe strips HTML tags for plain-text titles.
var weiboTagRe = regexp.MustCompile(`<[^>]*>`)

func plainWeiboText(htmlText string) string {
	return strings.TrimSpace(weiboTagRe.ReplaceAllString(htmlText, ""))
}

// WeiboUserHandler handles /weibo/user/:uid.
func WeiboUserHandler(c *ctxpkg.Context) (*models.Feed, error) {
	if err := requireWeiboCookies(); err != nil {
		return nil, err
	}
	uid := c.Param("uid")
	showRetweeted := routeutils.ParseBool(c.QueryParam("showRetweeted"), true)
	ctx := c.Parent()

	var container weiboContainerResp
	referer := fmt.Sprintf("https://m.weibo.cn/u/%s", uid)
	if err := weiboAPIProfile(referer).Fetch(
		"https://m.weibo.cn/api/container/getIndex?type=uid&value="+uid).
		GetJSON(ctx, c.Client(), &container); err != nil {
		return nil, err
	}
	if container.Data.UserInfo.ScreenName == "" {
		return nil, fmt.Errorf("weibo: no userInfo for uid %s (cookies may be expired)", uid)
	}
	containerID := "107603" + uid
	for _, tab := range container.Data.TabsInfo.Tabs {
		if tab.TabType == "weibo" && tab.ContainerID != "" {
			containerID = tab.ContainerID
			break
		}
	}

	var cards weiboCardsResp
	if err := weiboAPIProfile(referer).Fetch(
		"https://m.weibo.cn/api/container/getIndex?type=uid&value="+uid+"&containerid="+containerID).
		GetJSON(ctx, c.Client(), &cards); err != nil {
		return nil, err
	}
	if len(cards.Data.Cards) == 0 && cards.OK == 0 {
		return nil, fmt.Errorf("weibo: no cards for uid %s (cookies may be expired)", uid)
	}

	name := container.Data.UserInfo.ScreenName
	feed := routeutils.NewFeed(
		fmt.Sprintf("%s的微博", name),
		"https://m.weibo.cn/u/"+uid,
		container.Data.UserInfo.Desc,
	)
	for _, card := range cards.Data.Cards {
		mb := card.MBlog
		if mb == nil || (mb.Text == "" && mb.Bid == "") {
			continue
		}
		if !showRetweeted && mb.Retweeted != nil {
			continue
		}
		title := plainWeiboText(mb.Text)
		if title == "" {
			title = fmt.Sprintf("%s的微博 %s", name, mb.Bid)
		}
		if len([]rune(title)) > 60 {
			title = string([]rune(title)[:60]) + "…"
		}
		item := routeutils.NewItem(title, weiboStatusLink(uid, mb), buildWeiboDesc(mb), parseWeiboDate(mb.CreatedAt))
		if item == nil {
			continue
		}
		routeutils.SetItemAuthor(item, name, "", "")
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

// weiboStatusLink builds the canonical desktop permalink for a post.
func weiboStatusLink(uid string, mb *weiboMBlog) string {
	bid := mb.Bid
	if bid == "" {
		bid = mb.ID
	}
	return fmt.Sprintf("https://weibo.com/%s/%s", uid, bid)
}

// buildWeiboDesc renders the post HTML plus repost context and stats footer.
func buildWeiboDesc(mb *weiboMBlog) string {
	var b strings.Builder
	b.WriteString(weiboTextCleanup(mb.Text))
	if rt := mb.Retweeted; rt != nil {
		b.WriteString("<br/><blockquote>")
		b.WriteString(weiboTextCleanup(rt.Text))
		b.WriteString("</blockquote>")
	}
	fmt.Fprintf(&b, "<br/><small>转发 %d · 评论 %d · 赞 %d</small>",
		mb.RepostsCount, mb.CommentsCount, mb.AttitudesCount)
	return b.String()
}
