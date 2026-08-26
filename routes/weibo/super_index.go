package routes

import (
	"fmt"
	"net/url"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// weiboSuperIndexTypes mirrors upstream's super_index type table; "feed" is
// the default tab (最新评论).
var weiboSuperIndexTypes = []string{"feed", "sort_time", "hot_sort", "soul", "video", "album"}

type weiboSuperCard struct {
	CardType  int              `json:"card_type"`
	MBlog     *weiboMBlog      `json:"mblog"`
	CardGroup []weiboSuperCard `json:"card_group"`
}

type weiboSuperIndexResp struct {
	OK   int `json:"ok"`
	Data struct {
		PageInfo struct {
			PageTitle string `json:"page_title"`
		} `json:"pageInfo"`
		Cards []weiboSuperCard `json:"cards"`
	} `json:"data"`
}

// weiboSuperIndexRoute serves /weibo/super_index/:id — a 超话 (super topic)
// board via the m.weibo.cn "<id>_-_<type>" container API.
var weiboSuperIndexRoute = routeutils.RouteSpec{
	Path:        "super_index/:id",
	Name:        "超话",
	Example:     "weibo/super_index/1008084989d223732bf6f02f75ea30efad58a9",
	URL:         "https://weibo.com/p/1008084989d223732bf6f02f75ea30efad58a9/super_index",
	Maintainers: []string{"xihale"},
	Description: "Weibo super topic (超话) posts by container id; type 默认 feed(最新评论)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true, EnvDeps: []string{weiboCookiesEnv}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "超话 ID, 可在超话页 URL 的 containerid 中找到"),
		routeutils.OptionalParam("type", "soul 精华 / video 视频 / album 相册 / hot_sort 热门 / sort_time 最新帖子 / feed 最新评论(默认)"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  WeiboSuperIndexHandler,
}

// weiboSuperIndexTypeRoute registers the deeper /super_index/:id/:type shape
// (gin has no optional segments).
var weiboSuperIndexTypeRoute = func() routeutils.RouteSpec {
	clone := weiboSuperIndexRoute
	clone.Path = "super_index/:id/:type"
	return clone
}()

// WeiboSuperIndexHandler handles /weibo/super_index/:id[/:type].
func WeiboSuperIndexHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	typ := routeutils.ParseEnum(c.Param("type"), "feed", weiboSuperIndexTypes...)
	ctx := c.Parent()

	containerID := id + "_-_" + typ
	referer := fmt.Sprintf("%s/p/index?containerid=%s&luicode=10000011&lfid=%s",
		weiboMobileAPIBase, url.QueryEscape(id+"_-_soul"), url.QueryEscape(id+"_-_main"))

	var resp weiboSuperIndexResp
	if err := weiboAPIProfile(referer).Fetch(
		fmt.Sprintf("%s/api/container/getIndex?containerid=%s&luicode=10000011&lfid=%s",
			weiboMobileAPIBase, url.QueryEscape(containerID), url.QueryEscape(id+"_-_main"))).
		GetJSON(ctx, c.Client(), &resp); err != nil {
		if weiboIsLoginWallErr(err) {
			return nil, weiboAuthError(fmt.Sprintf("超话 %s", id), -100)
		}
		return nil, err
	}
	mblogs := collectWeiboSuperMBlogs(resp.Data.Cards)
	if resp.OK != 1 && len(mblogs) == 0 {
		return nil, weiboAuthError(fmt.Sprintf("超话 %s", id), resp.OK)
	}

	title := resp.Data.PageInfo.PageTitle
	displayTitle := title
	if displayTitle == "" {
		displayTitle = id
	}
	feed := routeutils.NewFeed(
		fmt.Sprintf("微博超话 - %s", displayTitle),
		fmt.Sprintf("https://weibo.com/p/%s/super_index", id),
		fmt.Sprintf("#%s# 的超话", displayTitle),
	)
	appendWeiboMBlogs(feed, "", "weibo-super-index-", mblogs)
	return feed, nil
}

// collectWeiboSuperMBlogs walks super-index cards, descending into nested
// card_group entries, and keeps only card_type=9 posts.
func collectWeiboSuperMBlogs(cards []weiboSuperCard) []*weiboMBlog {
	var out []*weiboMBlog
	for i := range cards {
		card := &cards[i]
		if card.CardType == 9 && card.MBlog != nil {
			out = append(out, card.MBlog)
		}
		out = append(out, collectWeiboSuperMBlogs(card.CardGroup)...)
	}
	return out
}
