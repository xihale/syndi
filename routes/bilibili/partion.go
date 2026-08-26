package routes

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

type bilibiliNewListResp struct {
	biliResp
	Data struct {
		Archives []biliVideo `json:"archives"`
	} `json:"data"`
}

// bilibiliZoneName derives a human readable zone name from the newest video's
// tname, falling back to "分区 <tid>".
func bilibiliZoneName(list []biliVideo, tid string) string {
	if len(list) > 0 && list[0].Tname != "" {
		return list[0].Tname
	}
	return "分区 " + tid
}

// fetchBilibiliZoneName best-effort zone name lookup via newlist.
func fetchBilibiliZoneName(c *ctxpkg.Context, tid string) (string, error) {
	var resp bilibiliNewListResp
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/newlist?ps=15&rid=%s&type=1&pn=1", url.QueryEscape(tid))
	if err := bilibiliJSONProfile().Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return "", err
	}
	if err := resp.Err(); err != nil {
		return "", err
	}
	return bilibiliZoneName(resp.Data.Archives, tid), nil
}

// BilibiliPartionHandler handles /bilibili/partion/:tid/:embed?
// Newest videos of a category via x/web-interface/newlist.
func BilibiliPartionHandler(c *ctxpkg.Context) (*models.Feed, error) {
	tid := c.Param("tid")
	if _, err := strconv.Atoi(tid); err != nil {
		return nil, fmt.Errorf("bilibili: invalid partition id %q", tid)
	}
	embed := bilibiliEmbedEnabled(c.Param("embed"))
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)
	ctx := c.Parent()

	var resp bilibiliNewListResp
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/newlist?ps=%d&rid=%s&type=1&pn=1", 30, url.QueryEscape(tid))
	if err := bilibiliJSONProfile().Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}

	name := bilibiliZoneName(resp.Data.Archives, tid)
	feed := routeutils.NewFeed(
		fmt.Sprintf("bilibili %s分区", name), bilibiliBaseURL,
		fmt.Sprintf("bilibili %s分区最新投稿", name),
	)
	appendPartionVideos(feed, resp.Data.Archives, limit, embed)
	return feed, nil
}

// appendPartionVideos maps newlist archives into items with partion GUIDs.
func appendPartionVideos(feed *models.Feed, list []biliVideo, limit int, embed bool) {
	routeutils.AppendMappedItems(feed, list, limit, func(v biliVideo) *models.Item {
		if v.Title == "" {
			return nil
		}
		desc := renderBilibiliUGCDescription(embed, v.Pic, html.EscapeString(v.Desc), v.Aid, 0, v.Bvid)
		item := routeutils.NewItem(v.Title, v.Link(), desc, time.Unix(v.Pubdate, 0))
		if item == nil {
			return nil
		}
		item.GUID = "bilibili-partion-" + v.ID()
		if v.Owner.Name != "" {
			routeutils.SetAuthor(item, v.Owner.Name, routeutils.WithAuthorURI(
				fmt.Sprintf("https://space.bilibili.com/%d", v.Owner.Mid)))
		}
		if v.Tname != "" {
			routeutils.SetCategories(item, v.Tname)
		}
		return item
	})
}

// --- partion/ranking ---

// bilibiliCateVideo is one hot-rank search result row.
// Count fields are flexible: the cate/search API sometimes returns numbers,
// sometimes digit strings.
type bilibiliCateVideo struct {
	ID        int64     `json:"id"`
	Bvid      string    `json:"bvid"`
	Title     string    `json:"title"`
	Desc      string    `json:"description"`
	Pic       string    `json:"pic"`
	Pubdate   string    `json:"pubdate"` // "2006-01-02 15:04:05" CST
	Author    string    `json:"author"`
	Mid       int64     `json:"mid"`
	Play      jsonInt64 `json:"play"`
	Favorites jsonInt64 `json:"favorites"`
	Review    jsonInt64 `json:"review"`
	VideoRev  jsonInt64 `json:"video_review"`
	Tag       string    `json:"tag"`
	Typename  string    `json:"typename"`
	Duration  int64     `json:"duration"`
}

type bilibiliCateSearchResp struct {
	biliResp
	Result []bilibiliCateVideo `json:"result"`
}

var bilibiliValidDays = map[int]bool{1: true, 3: true, 7: true, 30: true, 90: true, 120: true}

// BilibiliPartionRankingHandler handles /bilibili/partion/ranking/:tid/:days?
// Hot-rank videos of a category via s.search.bilibili.com cate/search.
func BilibiliPartionRankingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	tid := c.Param("tid")
	days := routeutils.ParseOptionalPositiveInt(c.Param("days"))
	n := 7
	if days != nil && bilibiliValidDays[*days] {
		n = *days
	}
	if _, err := strconv.Atoi(tid); err != nil {
		return nil, fmt.Errorf("bilibili: invalid partition id %q", tid)
	}
	ctx := c.Parent()

	name, err := fetchBilibiliZoneName(c, tid)
	if err != nil {
		return nil, fmt.Errorf("bilibili: resolve zone name: %w", err)
	}

	timeTo := time.Now()
	timeFrom := timeTo.AddDate(0, 0, -n)
	var resp bilibiliCateSearchResp
	apiURL := fmt.Sprintf(
		"https://s.search.bilibili.com/cate/search?main_ver=v3&search_type=video&view_type=hot_rank&cate_id=%s&time_from=%s&time_to=%s",
		url.QueryEscape(tid),
		timeFrom.Format("20060102"), timeTo.Format("20060102"),
	)
	if err := bilibiliJSONProfile().Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("bilibili %s 最热视频", name), bilibiliBaseURL,
		fmt.Sprintf("bilibili %s区最热视频（近%d天）", name, n),
	)
	routeutils.AppendMappedItems(feed, resp.Result, 0, mapCateVideo)
	return feed, nil
}

// mapCateVideo maps a hot-rank row into a feed item.
func mapCateVideo(r bilibiliCateVideo) *models.Item {
	if r.Title == "" || r.ID == 0 {
		return nil
	}
	link := videoLink(r.Bvid, r.ID)
	descText := html.EscapeString(r.Desc)
	if tag := html.EscapeString(r.Tag); tag != "" {
		if descText != "" {
			descText += " - "
		}
		descText += tag
	}
	desc := renderBilibiliUGCDescription(true, r.Pic, descText, r.ID, 0, r.Bvid)
	desc += fmt.Sprintf("<br/>Views: %d | Danmaku: %d | Favorites: %d | Comments: %d",
		r.Play, r.VideoRev, r.Favorites, r.Review)

	item := routeutils.NewItem(r.Title, link, desc, parseCSTDateTime(r.Pubdate))
	if item == nil {
		return nil
	}
	item.GUID = "bilibili-partion-ranking-" + strconv.FormatInt(r.ID, 10)
	if r.Author != "" {
		routeutils.SetAuthor(item, r.Author, routeutils.WithAuthorURI(
			fmt.Sprintf("https://space.bilibili.com/%d", r.Mid)))
	}
	if r.Typename != "" {
		routeutils.SetCategories(item, r.Typename)
	}
	return item
}

// parseCSTDateTime parses "2006-01-02 15:04:05" as Beijing time.
func parseCSTDateTime(raw string) time.Time {
	loc := time.FixedZone("CST", 8*3600)
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", raw, loc); err == nil {
		return t
	}
	return time.Time{}
}
