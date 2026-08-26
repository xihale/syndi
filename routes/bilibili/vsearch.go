package routes

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// bilibiliSearchVideo is one row of x/web-interface/search/type video results.
type bilibiliSearchVideo struct {
	Aid       int64  `json:"aid"`
	Bvid      string `json:"bvid"`
	Mid       int64  `json:"mid"`
	Author    string `json:"author"`
	Title     string `json:"title"`
	Desc      string `json:"description"`
	Pic       string `json:"pic"`
	Pubdate   int64  `json:"pubdate"`
	Duration  string `json:"duration"`
	Play      int64  `json:"play"`
	Favorites int64  `json:"favorites"`
	Review    int64  `json:"review"`       // comments
	VideoRev  int64  `json:"video_review"` // danmaku
	Tag       string `json:"tag"`
	Typename  string `json:"typename"`
	ArcURL    string `json:"arcurl"`
}

type bilibiliSearchTypeResp struct {
	biliResp
	Data struct {
		NumResults int64                 `json:"numResults"`
		Result     []bilibiliSearchVideo `json:"result"`
	} `json:"data"`
}

var bilibiliSearchOrders = []string{"totalrank", "click", "pubdate", "dm", "stow"}

// BilibiliVsearchHandler handles /bilibili/vsearch/:kw/:order?
// (+ optional embed/tid as query params for gin compatibility).
// Uses the public search API; a buvid3 cookie from finger/spi is required.
func BilibiliVsearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	kw := strings.TrimSpace(c.Param("kw"))
	if kw == "" {
		return nil, fmt.Errorf("bilibili: missing keyword")
	}
	order := routeutils.ParseEnum(c.Param("order"), "pubdate", bilibiliSearchOrders...)
	if raw := c.QueryParam("order"); raw != "" {
		order = routeutils.ParseEnum(raw, order, bilibiliSearchOrders...)
	}
	tid := firstNonEmpty(c.QueryParam("tid"), "0")
	if p := c.Param("tid"); p != "" {
		tid = p
	}
	embed := bilibiliEmbedEnabled(c.Param("embed"))

	cookie, err := bilibiliBuvidCookie(c)
	if err != nil {
		return nil, fmt.Errorf("bilibili: fetch buvid: %w", err)
	}

	apiURL := fmt.Sprintf(
		"https://api.bilibili.com/x/web-interface/search/type?search_type=video&highlight=1&keyword=%s&order=%s&tids=%s",
		urlSegment(kw), urlSegment(order), urlSegment(tid))
	var resp bilibiliSearchTypeResp
	if err := bilibiliJSONProfile().Cookie(cookie).
		Referer("https://search.bilibili.com/all?keyword="+urlSegment(kw)).
		Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}

	feedLink := fmt.Sprintf("https://search.bilibili.com/all?keyword=%s&order=%s", urlSegment(kw), urlSegment(order))
	return bilibiliVsearchFeed(kw, order, embed, feedLink, &resp), nil
}

// bilibiliVsearchFeed renders video search rows into a feed.
func bilibiliVsearchFeed(kw, order string, embed bool, feedLink string, resp *bilibiliSearchTypeResp) *models.Feed {
	feed := routeutils.NewFeed(fmt.Sprintf("%s - bilibili", kw), feedLink,
		fmt.Sprintf("Result from %s bilibili search, ordered by %s.", kw, order))

	routeutils.AppendMappedItems(feed, resp.Data.Result, 0, func(v bilibiliSearchVideo) *models.Item {
		if v.Title == "" || (v.Aid == 0 && v.Bvid == "") {
			return nil
		}
		title := stripEmTags(v.Title)
		link := videoLink(v.Bvid, v.Aid)
		if v.ArcURL != "" {
			link = v.ArcURL
		}
		desc := renderBilibiliUGCDescription(embed, bilibiliFeedImage(v.Pic),
			html.EscapeString(strings.ReplaceAll(v.Desc, "\n", "<br/>")), v.Aid, 0, v.Bvid)
		meta := fmt.Sprintf("<br/>Length: %s | AuthorID: %d<br/>Play: %d | Favorite: %d | Danmaku: %d | Comment: %d",
			html.EscapeString(v.Duration), v.Mid, v.Play, v.Favorites, v.VideoRev, v.Review)

		item := routeutils.NewItem(title, link, desc+meta, time.Unix(v.Pubdate, 0))
		if item == nil {
			return nil
		}
		gid := strconv.FormatInt(v.Aid, 10)
		if v.Bvid != "" {
			gid = v.Bvid
		}
		item.GUID = "bilibili-vsearch-" + gid
		if v.Author != "" {
			routeutils.SetAuthor(item, v.Author, routeutils.WithAuthorURI(
				fmt.Sprintf("https://space.bilibili.com/%d", v.Mid)))
		}
		var cats []string
		for _, t := range strings.Split(v.Tag, ",") {
			if t = strings.TrimSpace(stripEmTags(t)); t != "" {
				cats = append(cats, t)
			}
		}
		if v.Typename != "" {
			cats = append(cats, v.Typename)
		}
		routeutils.SetCategories(item, cats...)
		return item
	})
	return feed
}
