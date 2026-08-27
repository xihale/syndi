package routes

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// weiboKeywordRoute serves /weibo/keyword/:keyword — keyword search over
// m.weibo.cn (upstream containerid scheme "100103type=61&q=<kw>&t=0").
var weiboKeywordRoute = routeutils.RouteSpec{
	Path:        "keyword/:keyword",
	Name:        "关键词",
	Example:     "weibo/keyword/RSSHub",
	Maintainers: []string{"xihale"},
	Description: "Track Weibo posts mentioning a keyword via the m.weibo.cn search container API",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true, EnvDeps: []string{weiboCookiesEnv}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("keyword", "你想订阅的微博关键词"),
		routeutils.RequiredParam("routeParams", "预留的 RSSHub 风格参数片段, 兼容上游路径形状 (当前不解析开关)"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  WeiboKeywordHandler,
}

// weiboKeywordParamsRoute registers the deeper /keyword/:keyword/:routeParams
// shape so RSSHub-style URLs keep working (gin has no optional segments).
var weiboKeywordParamsRoute = func() routeutils.RouteSpec {
	clone := weiboKeywordRoute
	clone.Path = "keyword/:keyword/:routeParams"
	return clone
}()

// WeiboKeywordHandler handles /weibo/keyword/:keyword[/:routeParams].
func WeiboKeywordHandler(c *ctxpkg.Context) (*models.Feed, error) {
	keyword := strings.TrimSpace(c.Param("keyword"))
	if keyword == "" {
		return nil, fmt.Errorf("weibo: 关键词不能为空")
	}
	ctx := c.Parent()

	// Upstream encodes the keyword once and embeds it in an already-encoded
	// containerid value ("type%3D61%26q%3D<kw>%26t%3D0"), so keep exactly one
	// encoding level overall.
	enc := url.PathEscape(keyword)
	containerID := fmt.Sprintf("100103type%%3D61%%26q%%3D%s%%26t%%3D0", enc)
	referer := fmt.Sprintf("%s/p/searchall?containerid=100103type%%3D1%%26q%%3D%s", weiboMobileAPIBase, enc)

	var cards weiboCardsResp
	if err := weiboAPIProfile(referer).Fetch(
		fmt.Sprintf("%s/api/container/getIndex?containerid=%s", weiboMobileAPIBase, containerID)).
		GetJSON(ctx, c.Client(), &cards); err != nil {
		if weiboIsLoginWallErr(err) {
			return nil, weiboAuthError(fmt.Sprintf("关键词 %q 搜索", keyword), -100)
		}
		return nil, err
	}
	mblogs := weiboCardsToMBlogs(cards.Data.Cards)
	if cards.OK != 1 && len(mblogs) == 0 {
		return nil, weiboAuthError(fmt.Sprintf("关键词 %q 搜索", keyword), cards.OK)
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("又有人在微博提到%s了", keyword),
		fmt.Sprintf("https://s.weibo.com/weibo/%s&b=1&nodup=1", enc),
		fmt.Sprintf("又有人在微博提到%s了", keyword),
	)
	appendWeiboMBlogs(feed, "", "weibo-keyword-", weiboCardsToMBlogs(cards.Data.Cards))
	return feed, nil
}

// weiboCardsToMBlogs flattens plain container cards into their mblog payloads.
func weiboCardsToMBlogs(cards []weiboCard) []*weiboMBlog {
	var out []*weiboMBlog
	for i := range cards {
		if cards[i].MBlog != nil {
			out = append(out, cards[i].MBlog)
		}
	}
	return out
}
