package routes

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// --- live/room ---

type bilibiliLiveInfoResp struct {
	biliResp
	Data struct {
		UID        int64  `json:"uid"`
		RoomID     int64  `json:"room_id"`
		ShortID    int64  `json:"short_id"`
		LiveStatus int64  `json:"live_status"`
		Title      string `json:"title"`
		Desc       string `json:"description"`
		Keyframe   string `json:"keyframe"`
		LiveTime   string `json:"live_time"` // "2006-01-02 15:04:05" CST when live
		AreaName   string `json:"area_name"`
		ParentArea string `json:"parent_area_name"`
	} `json:"data"`
}

type bilibiliLiveMasterInfoResp struct {
	biliResp
	Data struct {
		Info struct {
			UID   int64  `json:"uid"`
			Uname string `json:"uname"`
			Face  string `json:"face"`
		} `json:"info"`
		FollowerNum int64 `json:"follower_num"`
	} `json:"data"`
}

func liveRoomURL(roomID int64) string {
	return "https://live.bilibili.com/" + strconv.FormatInt(roomID, 10)
}

// parseCSTTime parses "2006-01-02 15:04(:05)" timestamps in Beijing time.
func parseCSTTime(raw string) time.Time {
	loc := time.FixedZone("CST", 8*3600)
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, strings.TrimSpace(raw), loc); err == nil {
			return t
		}
	}
	return time.Time{}
}

// fetchBilibiliLiveMaster loads (and caches) the anchor profile of a room.
func fetchBilibiliLiveMaster(c *ctxpkg.Context, uid int64) (*bilibiliLiveMasterInfoResp, error) {
	var resp bilibiliLiveMasterInfoResp
	fetch := func() (*bilibiliLiveMasterInfoResp, error) {
		apiURL := fmt.Sprintf("https://api.live.bilibili.com/live_user/v1/Master/info?uid=%d", uid)
		if err := bilibiliJSONProfile().
			Referer("https://live.bilibili.com/").
			Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
			return nil, err
		}
		return &resp, resp.Err()
	}
	if c.Cache() == nil {
		return fetch()
	}
	v, err := c.CacheTryGet(fmt.Sprintf("bilibili-live-master-%d", uid), 24*time.Hour, func() (interface{}, error) {
		return fetch()
	})
	if err != nil {
		return nil, err
	}
	master, ok := v.(*bilibiliLiveMasterInfoResp)
	if !ok {
		return nil, fmt.Errorf("bilibili: invalid cached live master entry")
	}
	return master, nil
}

// BilibiliLiveRoomHandler handles /bilibili/live/room/:roomID.
// Short and long room ids both accepted via Room/get_info; the anchor name
// comes from Master/info (cached).
func BilibiliLiveRoomHandler(c *ctxpkg.Context) (*models.Feed, error) {
	raw := c.Param("roomID")
	roomIDInt, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || roomIDInt <= 0 {
		return nil, fmt.Errorf("bilibili: invalid room id %q", raw)
	}
	ctx := c.Parent()

	var info bilibiliLiveInfoResp
	apiURL := fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/get_info?room_id=%d&from=room", roomIDInt)
	if err := bilibiliJSONProfile().
		Referer(liveRoomURL(roomIDInt)).
		Fetch(apiURL).GetJSON(ctx, c.Client(), &info); err != nil {
		return nil, err
	}
	if err := info.Err(); err != nil {
		return nil, err
	}

	uname := ""
	if info.Data.UID != 0 {
		if master, merr := fetchBilibiliLiveMaster(c, info.Data.UID); merr == nil {
			uname = master.Data.Info.Uname
		}
	}
	displayName := firstNonEmpty(uname, fmt.Sprintf("直播间 %s", raw))
	return bilibiliLiveRoomFeed(displayName, uname, raw, &info), nil
}

// bilibiliLiveRoomFeed renders the room status feed (empty when offline).
func bilibiliLiveRoomFeed(displayName, uname, requestedID string, info *bilibiliLiveInfoResp) *models.Feed {
	roomID := info.Data.RoomID
	if roomID == 0 && requestedID != "" {
		if parsed, err := strconv.ParseInt(requestedID, 10, 64); err == nil {
			roomID = parsed
		}
	}
	link := liveRoomURL(roomID)
	feed := routeutils.NewFeed(
		fmt.Sprintf("%s 直播间开播状态", displayName), link,
		fmt.Sprintf("%s 直播间开播状态", displayName),
	)
	if info.Data.LiveStatus != 1 {
		return feed // offline rooms produce an empty feed like upstream allowEmpty
	}

	title := strings.TrimSpace(info.Data.Title + " " + info.Data.LiveTime)
	desc := ""
	if info.Data.Keyframe != "" {
		desc += fmt.Sprintf(`<img src="%s"/><br/>`, html.EscapeString(bilibiliFeedImage(info.Data.Keyframe)))
	}
	if info.Data.Desc != "" {
		desc += sanitizeLiveDescription(info.Data.Desc)
	}

	item := routeutils.NewItem(title, link, desc, parseCSTTime(info.Data.LiveTime))
	if item == nil {
		return feed
	}
	item.GUID = fmt.Sprintf("bilibili-live-room-%d-%s", roomID, info.Data.LiveTime)
	if uname != "" {
		routeutils.SetAuthor(item, uname, routeutils.WithAuthorURI(
			fmt.Sprintf("https://space.bilibili.com/%d", info.Data.UID)))
	}
	if area := firstNonEmpty(info.Data.AreaName, info.Data.ParentArea); area != "" {
		routeutils.SetCategories(item, area)
	}
	routeutils.AddItem(feed, item)
	return feed
}

// liveDescriptionTags are allowed in server-provided room descriptions.
var liveDescriptionTags = []string{"p", "br", "img", "a", "span", "b", "i"}

// sanitizeLiveDescription strips scripts/styles/handlers from trusted-ish
// upstream HTML while keeping basic formatting.
func sanitizeLiveDescription(raw string) string {
	out, err := routeutils.SanitizeHTML(raw, liveDescriptionTags, nil)
	if err != nil {
		return html.EscapeString(routeutils.ExtractText(raw))
	}
	return out
}

// --- live/search ---

type bilibiliLiveSearchResp struct {
	biliResp
	Data struct {
		Result struct {
			// live_room is a plain array in real responses (despite some docs
			// showing {total, list}).
			LiveRoom []struct {
				RoomID      int64  `json:"roomid"`
				UID         int64  `json:"uid"`
				Uname       string `json:"uname"`
				Title       string `json:"title"`
				Cover       string `json:"cover"`
				UserCover   string `json:"user_cover"`
				Tags        string `json:"tags"`
				LiveTime    string `json:"live_time"`
				WatchedShow struct {
					TextLarge string `json:"text_large"`
					Num       int64  `json:"num"`
				} `json:"watched_show"`
			} `json:"live_room"`
		} `json:"result"`
	} `json:"data"`
}

// emTagRe removes search-highlight wrappers such as <em class="keyword">.
var emTagRe = regexp.MustCompile(`<[^>]*>`)

func stripEmTags(s string) string {
	return emTagRe.ReplaceAllString(s, "")
}

// BilibiliLiveSearchHandler handles /bilibili/live/search/:key.
func BilibiliLiveSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	key := strings.TrimSpace(c.Param("key"))
	if key == "" {
		return nil, fmt.Errorf("bilibili: missing search keyword")
	}

	var resp bilibiliLiveSearchResp
	apiURL := "https://api.bilibili.com/x/web-interface/search/type?search_type=live&page=1&keyword=" + urlSegment(key)
	profile := bilibiliJSONProfile().
		Referer("https://search.bilibili.com/live?keyword=" + urlSegment(key))
	if cookie, cerr := bilibiliBuvidCookie(c); cerr == nil {
		profile = profile.Cookie(cookie)
	}
	if err := profile.Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return bilibiliLiveSearchFeed(key, &resp), nil
}

// bilibiliLiveSearchFeed renders live search result rows.
func bilibiliLiveSearchFeed(key string, resp *bilibiliLiveSearchResp) *models.Feed {
	feedLink := "https://search.bilibili.com/live?keyword=" + urlSegment(key)
	feed := routeutils.NewFeed(fmt.Sprintf("%s - bilibili 直播搜索", key), feedLink,
		fmt.Sprintf("bilibili 与 %s 相关的直播间", key))

	for _, r := range resp.Data.Result.LiveRoom {
		if r.RoomID == 0 {
			continue
		}
		title := strings.TrimSpace(stripEmTags(r.Title))
		if title == "" {
			continue
		}
		cover := bilibiliFeedImage(firstNonEmpty(r.UserCover, r.Cover))
		desc := fmt.Sprintf("Up: %s | Watching: %s",
			html.EscapeString(stripEmTags(r.Uname)),
			html.EscapeString(stripEmTags(r.WatchedShow.TextLarge)))
		if cover != "" {
			desc += fmt.Sprintf(`<br/><img src="%s"/>`, cover)
		}
		item := routeutils.NewItem(title, liveRoomURL(r.RoomID), desc, parseCSTTime(r.LiveTime))
		if item == nil {
			continue
		}
		item.GUID = fmt.Sprintf("bilibili-live-search-%d", r.RoomID)
		if r.Uname != "" {
			routeutils.SetAuthor(item, stripEmTags(r.Uname), routeutils.WithAuthorURI(
				fmt.Sprintf("https://space.bilibili.com/%d", r.UID)))
		}
		if tags := stripEmTags(r.Tags); tags != "" {
			routeutils.SetCategories(item, strings.Split(tags, ",")...)
		}
		routeutils.AddItem(feed, item)
	}
	return feed
}

// --- weekly 每周必看 ---

type bilibiliWeeklySeriesResp struct {
	biliResp
	Data []struct {
		Number int64  `json:"number"`
		Name   string `json:"name"`
		Status int64  `json:"status"`
	} `json:"data"`
}

type bilibiliWeeklySelectedResp struct {
	biliResp
	Data struct {
		List []struct {
			Title           string `json:"title"`
			Param           string `json:"param"` // aid when goto=av
			Bvid            string `json:"bvid"`
			Cover           string `json:"cover"`
			RcmdReason      string `json:"rcmd_reason"`
			CoverRightText1 string `json:"cover_right_text_1"`
		} `json:"list"`
	} `json:"data"`
}

// BilibiliWeeklyHandler handles /bilibili/weekly/:embed?
func BilibiliWeeklyHandler(c *ctxpkg.Context) (*models.Feed, error) {
	embed := bilibiliEmbedEnabled(c.Param("embed"))
	ctx := c.Parent()

	var series bilibiliWeeklySeriesResp
	if err := bilibiliJSONProfile().
		Referer("https://www.bilibili.com/h5/weekly-recommend").
		Fetch("https://app.bilibili.com/x/v2/show/popular/selected/series?type=weekly_selected").
		GetJSON(ctx, c.Client(), &series); err != nil {
		return nil, err
	}
	if err := series.Err(); err != nil {
		return nil, err
	}
	if len(series.Data) == 0 {
		return nil, fmt.Errorf("bilibili: no weekly series available")
	}
	current := series.Data[0]

	var resp bilibiliWeeklySelectedResp
	apiURL := fmt.Sprintf("https://app.bilibili.com/x/v2/show/popular/selected?type=weekly_selected&number=%d", current.Number)
	if err := bilibiliJSONProfile().
		Referer(fmt.Sprintf("https://www.bilibili.com/h5/weekly-recommend?num=%d&navhide=1", current.Number)).
		Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return bilibiliWeeklyFeed(&current, &resp, embed), nil
}

// bilibiliWeeklyFeed renders the weekly selection items.
func bilibiliWeeklyFeed(current *struct {
	Number int64  `json:"number"`
	Name   string `json:"name"`
	Status int64  `json:"status"`
}, resp *bilibiliWeeklySelectedResp, embed bool) *models.Feed {
	link := "https://www.bilibili.com/h5/weekly-recommend"
	weekName := firstNonEmpty(current.Name, "每周必看")
	feed := routeutils.NewFeed("B站每周必看", link, weekName)

	for _, wv := range resp.Data.List {
		if wv.Title == "" {
			continue
		}
		aid, _ := strconv.ParseInt(wv.Param, 10, 64)

		var descParts []string
		summary := strings.TrimSpace(strings.Join([]string{weekName, wv.Title}, " "))
		if wv.RcmdReason != "" {
			summary = summary + " - " + wv.RcmdReason
		}
		descParts = append(descParts, html.EscapeString(summary))
		if wv.CoverRightText1 != "" {
			descParts = append(descParts, "Duration: "+html.EscapeString(wv.CoverRightText1))
		}
		desc := renderBilibiliUGCDescription(embed, bilibiliFeedImage(wv.Cover),
			strings.Join(descParts, "<br/>"), aid, 0, wv.Bvid)

		item := routeutils.NewItem(wv.Title, videoLink(wv.Bvid, aid), desc, time.Time{})
		if item == nil {
			continue
		}
		gid := wv.Bvid
		if gid == "" {
			gid = wv.Param
		}
		item.GUID = "bilibili-weekly-" + gid
		routeutils.AddItem(feed, item)
	}
	return feed
}
