package routes

import (
	"fmt"
	"html"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

type bilibiliViewResp struct {
	biliResp
	Data struct {
		Aid     int64  `json:"aid"`
		Bvid    string `json:"bvid"`
		Title   string `json:"title"`
		Desc    string `json:"desc"`
		Pic     string `json:"pic"`
		Pubdate int64  `json:"pubdate"`
		Tname   string `json:"tname"`
		Pages   []struct {
			CID      int64  `json:"cid"`
			Page     int64  `json:"page"`
			Part     string `json:"part"`
			Duration int64  `json:"duration"`
		} `json:"pages"`
	} `json:"data"`
}

// bilibiliNormalizeVideoID accepts a BV id or numeric av id; it returns
// (bvid, aid) where exactly one is populated.
func bilibiliNormalizeVideoID(raw string) (string, int64) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToUpper(raw), "BV") {
		return raw, 0
	}
	trimmed := strings.TrimPrefix(raw, "av")
	aid, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || aid <= 0 {
		return raw, 0
	}
	return "", aid
}

func videoLink(bvid string, aid int64) string {
	if bvid != "" {
		return bilibiliBaseURL + "/video/" + bvid
	}
	return fmt.Sprintf("%s/video/av%d", bilibiliBaseURL, aid)
}

// fetchBilibiliView loads x/web-interface/view metadata for the given id.
func fetchBilibiliView(c *ctxpkg.Context, bvid string, aid int64) (*bilibiliViewResp, error) {
	query := "bvid=" + url.QueryEscape(bvid)
	if bvid == "" {
		query = "aid=" + strconv.FormatInt(aid, 10)
	}
	link := videoLink(bvid, aid)
	var resp bilibiliViewResp
	apiURL := "https://api.bilibili.com/x/web-interface/view?" + query
	if err := bilibiliJSONProfile().Referer(link).Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BilibiliVideoPageHandler handles /bilibili/video/page/:bvid/:embed?
// Videos episodes list via the public view API.
func BilibiliVideoPageHandler(c *ctxpkg.Context) (*models.Feed, error) {
	bvid, aid := bilibiliNormalizeVideoID(c.Param("bvid"))
	embed := bilibiliEmbedEnabled(c.Param("embed"))
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 10, 100)

	resp, err := fetchBilibiliView(c, bvid, aid)
	if err != nil {
		return nil, err
	}
	return bilibiliVideoPageFeed(resp, embed, limit), nil
}

// bilibiliVideoPageFeed renders the episode list of a view payload
// (newest-first like upstream).
func bilibiliVideoPageFeed(resp *bilibiliViewResp, embed bool, limit int) *models.Feed {
	link := videoLink(resp.Data.Bvid, resp.Data.Aid)
	feed := routeutils.NewFeed(
		fmt.Sprintf("视频 %s 的选集列表", resp.Data.Title),
		link, fmt.Sprintf("视频 %s 的视频选集列表", resp.Data.Title),
	)

	pages := resp.Data.Pages
	sort.SliceStable(pages, func(i, j int) bool { return pages[i].Page > pages[j].Page })

	routeutils.AppendMappedItems(feed, pages, limit, func(p struct {
		CID      int64  `json:"cid"`
		Page     int64  `json:"page"`
		Part     string `json:"part"`
		Duration int64  `json:"duration"`
	}) *models.Item {
		desc := renderBilibiliUGCDescription(embed,
			bilibiliFeedImage(resp.Data.Pic),
			html.EscapeString(p.Part+" - "+resp.Data.Title),
			resp.Data.Aid, p.CID, resp.Data.Bvid)
		item := routeutils.NewItem(p.Part, fmt.Sprintf("%s?p=%d", link, p.Page), desc, time.Time{})
		if item == nil {
			return nil
		}
		item.GUID = fmt.Sprintf("bilibili-video-page-%d-%d", resp.Data.Aid, p.Page)
		if resp.Data.Tname != "" {
			routeutils.SetCategories(item, resp.Data.Tname)
		}
		return item
	})
	return feed
}

// --- video/reply ---

type bilibiliReplyResp struct {
	biliResp
	Data struct {
		Cursor struct {
			AllCount int64 `json:"all_count"`
			Mode     int64 `json:"mode"`
		} `json:"cursor"`
		Replies []struct {
			RPID   int64 `json:"rpid"`
			Member struct {
				Uname string `json:"uname"`
			} `json:"member"`
			Content struct {
				Message string `json:"message"`
			} `json:"content"`
			CTime int64 `json:"ctime"`
			Count int64 `json:"count"`
		} `json:"replies"`
	} `json:"data"`
}

// BilibiliVideoReplyHandler handles /bilibili/video/reply/:bvid.
// Comments come from x/v2/reply/main; mode defaults to hot (3).
func BilibiliVideoReplyHandler(c *ctxpkg.Context) (*models.Feed, error) {
	bvid, aid := bilibiliNormalizeVideoID(c.Param("bvid"))
	mode := c.QueryParam("mode")
	if mode == "" {
		mode = "3" // hot comments, what the web player shows by default
	}

	title := ""
	if bvid != "" || aid != 0 {
		if resp, err := fetchBilibiliView(c, bvid, aid); err == nil {
			title = resp.Data.Title
			if bvid == "" && resp.Data.Bvid != "" {
				bvid = resp.Data.Bvid
			}
			if aid == 0 {
				aid = resp.Data.Aid
			}
		}
	}
	if aid == 0 {
		return nil, fmt.Errorf("bilibili: cannot resolve oid for reply of %s", c.Param("bvid"))
	}

	link := videoLink(bvid, aid)
	var resp bilibiliReplyResp
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/v2/reply/main?type=1&oid=%d&mode=%s", aid, url.QueryEscape(mode))
	// The gaia gateway risk-controls anonymous comment reads (-352); a buvid
	// pair from finger/spi is enough without login.
	profile := bilibiliJSONProfile().Referer(link)
	if cookie, cerr := bilibiliBuvidCookie(c); cerr == nil {
		profile = profile.Cookie(cookie)
	}
	if err := profile.Fetch(apiURL).
		GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}

	feedTitle := title
	if feedTitle == "" {
		feedTitle = strconv.FormatInt(aid, 10)
	}
	return bilibiliReplyFeed(feedTitle, link, aid, &resp), nil
}

// bilibiliReplyFeed renders comment items from a reply/main payload.
func bilibiliReplyFeed(title, link string, aid int64, resp *bilibiliReplyResp) *models.Feed {
	feed := routeutils.NewFeed(
		fmt.Sprintf("%s 的评论", title), link,
		fmt.Sprintf("%s 的评论", title),
	)
	for _, r := range resp.Data.Replies {
		msg := strings.TrimSpace(r.Content.Message)
		uname := strings.TrimSpace(r.Member.Uname)
		if msg == "" {
			continue
		}
		text := uname + ": " + msg
		description := html.EscapeString(text)
		if r.Count > 0 {
			description += fmt.Sprintf("<br/><i>%d 条回复</i>", r.Count)
		}
		item := routeutils.NewItem(
			html.EscapeString(text),
			fmt.Sprintf("%s#reply%d", link, r.RPID),
			description,
			time.Unix(r.CTime, 0),
		)
		if item == nil {
			continue
		}
		item.GUID = fmt.Sprintf("bilibili-video-reply-%d-%d", aid, r.RPID)
		if uname != "" {
			routeutils.SetAuthor(item, uname)
		}
		routeutils.AddItem(feed, item)
	}
	return feed
}
