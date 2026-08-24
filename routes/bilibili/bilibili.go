package routes

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const bilibiliBaseURL = "https://www.bilibili.com"

// bilibiliJSONProfile returns the shared disguise profile for bilibili APIs.
func bilibiliJSONProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9").JSONAccept().
		Referer("https://www.bilibili.com/")
}

var bilibiliPopularRoute = routeutils.RouteSpec{
	Path:        "popular",
	Name:        "Popular Videos",
	Example:     "bilibili/popular",
	Maintainers: []string{"xihale"},
	Description: "Bilibili trending popular videos",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Maximum number of items, default 20, max 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  BilibiliPopularHandler,
}

var bilibiliHotSearchRoute = routeutils.RouteSpec{
	Path:        "hot-search",
	Name:        "Hot Search",
	Example:     "bilibili/hot-search",
	Maintainers: []string{"xihale"},
	Description: "Bilibili hot search keywords",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    15 * time.Minute,
	Handler:     BilibiliHotSearchHandler,
}

var bilibiliRankingRoute = routeutils.RouteSpec{
	Path:        "ranking",
	Name:        "Ranking",
	Example:     "bilibili/ranking",
	Maintainers: []string{"xihale"},
	Description: "Bilibili overall ranking board",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Maximum number of items, default 50, max 100"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  BilibiliRankingHandler,
}

var bilibiliRankingZoneRoute = routeutils.RouteSpec{
	Path:        "ranking/:rid",
	Name:        "Ranking by Zone",
	Example:     "bilibili/ranking/1",
	Maintainers: []string{"xihale"},
	Description: "Bilibili ranking board for a zone, rid 0-34 (1 anime, 3 music, 5 game)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("rid", "Ranking zone id; e.g. 1 anime, 3 music, 5 game"),
		routeutils.OptionalParam("limit", "Maximum number of items, default 50, max 100"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  BilibiliRankingHandler,
}

// BilibiliPopularHandler handles /bilibili/popular
func BilibiliPopularHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()

	var resp biliListResp
	if err := bilibiliJSONProfile().Fetch("https://api.bilibili.com/x/web-interface/popular?ps=50").
		GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("bilibili 综合热门", bilibiliBaseURL, "bilibili 综合热门")
	appendBilibiliVideos(feed, resp.Data.List, limit)
	return feed, nil
}

// BilibiliHotSearchHandler handles /bilibili/hot-search
func BilibiliHotSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()

	var resp biliHotSearchResp
	if err := bilibiliJSONProfile().Fetch("https://api.bilibili.com/x/web-interface/search/square?limit=50&platform=pc").
		GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}

	trending := resp.Data.Trending
	feed := routeutils.NewFeed(trending.Title, bilibiliBaseURL, "bilibili 热搜")
	for _, entry := range trending.List {
		if entry.Keyword == "" {
			continue
		}
		desc := html.EscapeString(entry.Keyword)
		if entry.Icon != "" {
			desc += fmt.Sprintf(`<br/><img src="%s" alt=""/>`, html.EscapeString(entry.Icon))
		}
		link := fmt.Sprintf("https://search.bilibili.com/all?keyword=%s&from_source=webtop_search", url.QueryEscape(entry.Keyword))
		item := routeutils.NewItem(entry.Keyword, link, desc, time.Time{})
		if item == nil {
			continue
		}
		item.GUID = "bilibili-hot-" + strconv.FormatInt(entry.HeatScore, 10) + "-" + entry.Keyword
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

// BilibiliRankingHandler handles /bilibili/ranking/:rid?
func BilibiliRankingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	rid := routeutils.ParsePositiveInt(c.Param("rid"), 1, 34) - 1
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 50, 100)
	ctx := c.Parent()

	apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/ranking/v2?rid=%d&type=all", rid)
	var resp biliListResp
	if err := bilibiliJSONProfile().Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("bilibili 排行榜", bilibiliBaseURL+"/v/popular/rank/all", "bilibili 排行榜")
	appendBilibiliVideos(feed, resp.Data.List, limit)
	return feed, nil
}

// appendBilibiliVideos maps video records into feed items.
func appendBilibiliVideos(feed *models.Feed, list []biliVideo, limit int) {
	routeutils.AppendMappedItems(feed, list, limit, func(v biliVideo) *models.Item {
		if v.Title == "" {
			return nil
		}
		link := v.Link()
		desc := ""
		if v.Pic != "" {
			desc += fmt.Sprintf(`<img src="%s" alt=""/><br/>`, html.EscapeString(v.Pic))
		}
		if v.Desc != "" {
			desc += html.EscapeString(v.Desc) + "<br/>"
		}
		desc += fmt.Sprintf("Author: %s | Category: %s | Views: %d | Danmaku: %d | Likes: %d",
			html.EscapeString(v.Owner.Name), html.EscapeString(v.Tname),
			v.Stat.View, v.Stat.Danmaku, v.Stat.Like)

		item := routeutils.NewItem(v.Title, link, desc, time.Unix(v.Pubdate, 0))
		if item == nil {
			return nil
		}
		if v.Aid != 0 {
			item.GUID = "bilibili-video-" + v.ID()
		}
		if v.Owner.Name != "" {
			routeutils.SetAuthor(item, v.Owner.Name, routeutils.WithAuthorURI(
				fmt.Sprintf("%s/%d", "https://space.bilibili.com", v.Owner.Mid)))
		}
		if v.Tname != "" {
			routeutils.SetCategories(item, v.Tname)
		}
		return item
	})
}

type biliResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Err converts a non-zero upstream code into a Go error.
func (r *biliResp) Err() error {
	if r.Code != 0 {
		return fmt.Errorf("bilibili api error %d: %s", r.Code, r.Message)
	}
	return nil
}

type biliListResp struct {
	biliResp
	Data struct {
		List []biliVideo `json:"list"`
	} `json:"data"`
}

type biliVideo struct {
	Aid     int64  `json:"aid"`
	Bvid    string `json:"bvid"`
	Title   string `json:"title"`
	Desc    string `json:"desc"`
	Pic     string `json:"pic"`
	Pubdate int64  `json:"pubdate"`
	Tname   string `json:"tname"`
	Owner   struct {
		Mid  int64  `json:"mid"`
		Name string `json:"name"`
		Face string `json:"face"`
	} `json:"owner"`
	Stat struct {
		View     int64 `json:"view"`
		Danmaku  int64 `json:"danmaku"`
		Reply    int64 `json:"reply"`
		Favorite int64 `json:"favorite"`
		Coin     int64 `json:"coin"`
		Share    int64 `json:"share"`
		Like     int64 `json:"like"`
	} `json:"stat"`
}

// ID returns the stable upstream id of the video.
func (v *biliVideo) ID() string {
	if v.Bvid != "" {
		return v.Bvid
	}
	return strconv.FormatInt(v.Aid, 10)
}

// Link builds the canonical watch URL.
func (v *biliVideo) Link() string {
	if v.Bvid != "" {
		return bilibiliBaseURL + "/video/" + v.Bvid
	}
	return bilibiliBaseURL + "/video/av" + strconv.FormatInt(v.Aid, 10)
}

type biliHotSearchResp struct {
	biliResp
	Data struct {
		Trending struct {
			Title string `json:"title"`
			List  []struct {
				Keyword   string `json:"keyword"`
				ShowName  string `json:"show_name"`
				Icon      string `json:"icon"`
				HeatScore int64  `json:"heat_score"`
			} `json:"list"`
		} `json:"trending"`
	} `json:"data"`
}
