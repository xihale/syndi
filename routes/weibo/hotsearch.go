package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// weiboHotSearchRoute serves /weibo/hotsearch — the trending topic board.
var weiboHotSearchRoute = routeutils.RouteSpec{
	Path:        "hotsearch",
	Name:        "Hot Search",
	Example:     "weibo/hotsearch",
	Maintainers: []string{"xihale"},
	Description: "Weibo realtime hot search board (热搜榜)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true, EnvDeps: []string{weiboCookiesEnv}},
	Parameters:  nil,
	CacheTTL:    10 * time.Minute,
	Handler:     WeiboHotSearchHandler,
}

type weiboHotResp struct {
	OK   int `json:"ok"`
	Data struct {
		Cards []struct {
			CardGroup []struct {
				Desc        string `json:"desc"`
				DescExpend  string `json:"desc_expend"`
				Scheme      string `json:"scheme"`
				WordScheme  string `json:"word_scheme"`
				IsAd        int    `json:"is_ad"`
				RawHotValue string `json:"raw_hot"`
			} `json:"card_group"`
		} `json:"cards"`
	} `json:"data"`
}

// WeiboHotSearchHandler handles /weibo/hotsearch.
func WeiboHotSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	if err := requireWeiboCookies(); err != nil {
		return nil, err
	}
	ctx := c.Parent()

	var resp weiboHotResp
	url := "https://m.weibo.cn/api/container/getIndex?containerid=106003type%3D25%26t%3D3%26disable_hot%3D1%26filter_type%3Drealtimehot&title=%E5%BE%AE%E5%8D%9A%E7%83%AD%E6%90%9C&extparam=filter_type%3Drealtimehot%26mi_cid%3D100103%26pos%3D0_0%26c_type%3D30%26display_time%3D1540538388&luicode=10000011&lfid=231583"
	if err := weiboAPIProfile("https://s.weibo.com/top/summary?cate=realtimehot").
		Fetch(url).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.OK == 0 && len(resp.Data.Cards) == 0 {
		return nil, fmt.Errorf("weibo: hot search unavailable (cookies may be expired)")
	}

	feed := routeutils.NewFeed(
		"微博热搜榜",
		"https://s.weibo.com/top/summary?cate=realtimehot",
		"微博实时热搜榜",
	)
	for _, card := range resp.Data.Cards {
		for _, entry := range card.CardGroup {
			title := strings.TrimSpace(entry.Desc)
			if title == "" || entry.IsAd != 0 {
				continue
			}
			link := entry.Scheme
			if link == "" {
				continue
			}
			link = strings.ReplaceAll(link, "https://m.weibo.cn", "https://s.weibo.cn")
			desc := title
			if entry.DescExpend != "" {
				desc = fmt.Sprintf("%s<br/><small>%s</small>", htmlEscapeText(title), htmlEscapeText(entry.DescExpend))
			}
			// The hot board exposes no per-entry timestamps; leave PubDate zero.
			item := routeutils.NewItem(fmt.Sprintf("%d. %s", len(feed.Items)+1, title), link, desc, time.Time{})
			routeutils.AddItem(feed, item)
		}
	}
	return feed, nil
}

// weiboSearchHotRoute serves /weibo/search/hot as a thin alias of the
// hotsearch board: both paths hit the identical m.weibo.cn realtime-hot
// container upstream, so they share one handler (upstream's fulltext digest
// expansion mode is not ported).
var weiboSearchHotRoute = routeutils.RouteSpec{
	Path:        "search/hot",
	Name:        "热搜榜（search/hot）",
	Example:     "weibo/search/hot",
	Maintainers: []string{"xihale"},
	Description: "RSSHub-style alias of /weibo/hotsearch; same board, same items. 上游的 fulltext 摘要展开模式未实现",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true, EnvDeps: []string{weiboCookiesEnv}},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("fulltext", "仅兼容上游路径形状, 当前忽略并返回榜单本体"),
	},
	CacheTTL: 10 * time.Minute,
	Handler:  WeiboHotSearchHandler,
}

// weiboSearchHotFulltextRoute registers the deeper /search/hot/:fulltext
// shape so RSSHub-style URLs keep working (gin has no optional segments).
var weiboSearchHotFulltextRoute = func() routeutils.RouteSpec {
	clone := weiboSearchHotRoute
	clone.Path = "search/hot/:fulltext"
	return clone
}()

func htmlEscapeText(s string) string {
	repl := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return repl.Replace(s)
}
