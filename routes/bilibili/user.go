package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// disguiseProfileForSpace builds an XHR-like profile for space-bound APIs.
func disguiseProfileForSpace(uid string) *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9").JSONAccept().
		Referer(spaceURL(uid)+"/video").
		WithHeader("Origin", "https://space.bilibili.com")
}

// jsonInt64 accepts either a JSON number or string-encoded number
// (the dynamic API mixes both).
type jsonInt64 int64

// Int64 returns the numeric value.
func (j jsonInt64) Int64() int64 { return int64(j) }

// UnmarshalJSON tolerates numbers and quoted numbers including null.
func (j *jsonInt64) UnmarshalJSON(data []byte) error {
	s := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if s == "" || s == "null" {
		*j = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid number %q", s)
	}
	*j = jsonInt64(v)
	return nil
}

// unmarshalJSONString decodes an embedded JSON string payload.
func unmarshalJSONString(raw string, target interface{}) error {
	if raw == "" {
		return fmt.Errorf("empty json string")
	}
	return json.Unmarshal([]byte(raw), target)
}

// --- user/video: UP 主投稿 ---

type bilibiliUserVideo struct {
	Aid         int64  `json:"aid"`
	Bvid        string `json:"bvid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Pic         string `json:"pic"`
	Created     int64  `json:"created"`
	Length      string `json:"length"`
	Author      string `json:"author"`
	Mid         int64  `json:"mid"`
	Comment     int64  `json:"comment"`
}

type bilibiliArcSearchResp struct {
	biliResp
	Data struct {
		List struct {
			Vlist []bilibiliUserVideo `json:"vlist"`
		} `json:"list"`
	} `json:"data"`
}

// BilibiliUserVideoHandler handles /bilibili/user/video/:uid/:embed?
// Signed with wbi (img/sub key from the public nav API), mirroring upstream.
func BilibiliUserVideoHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return bilibiliUserVideoFeed(c)
}

func bilibiliUserVideoFeed(c *ctxpkg.Context) (*models.Feed, error) {
	uid := c.Param("uid")
	if uid == "" {
		return nil, fmt.Errorf("bilibili: missing uid")
	}
	embed := bilibiliEmbedEnabled(c.Param("embed"))
	ctx := c.Parent()

	imgKey, subKey, err := bilibiliWbiKeys(c)
	if err != nil {
		return nil, fmt.Errorf("bilibili: fetch wbi keys: %w", err)
	}

	params := url.Values{}
	params.Set("mid", uid)
	params.Set("ps", "30")
	params.Set("tid", "0")
	params.Set("pn", "1")
	params.Set("keyword", "")
	params.Set("order", "pubdate")
	params.Set("platform", "web")
	params.Set("web_location", "1550101")
	params.Set("order_avoided", "true")
	params.Set("dm_img_list", bilibiliDmImgList())
	signed := bilibiliSignWbi(params, imgKey, subKey, time.Now().Unix())

	profile := disguiseProfileForSpace(uid)
	if cookie, cerr := bilibiliBuvidCookie(c); cerr == nil {
		profile = profile.Cookie(cookie)
	}
	var resp bilibiliArcSearchResp
	apiURL := "https://api.bilibili.com/x/space/wbi/arc/search?" + signed.Encode()
	if err := profile.Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s 的 bilibili 投稿视频", uid),
		spaceURL(uid), "bilibili 用户投稿视频",
	)
	appendUserVideos(feed, resp.Data.List.Vlist, embed, uid)
	return feed, nil
}

func spaceURL(uid string) string {
	return "https://space.bilibili.com/" + uid
}

// appendUserVideos maps a space archive list into feed items.
func appendUserVideos(feed *models.Feed, list []bilibiliUserVideo, embed bool, uid string) {
	routeutils.AppendMappedItems(feed, list, 0, func(v bilibiliUserVideo) *models.Item {
		if v.Title == "" {
			return nil
		}
		link := bilibiliBaseURL + "/video/"
		if v.Bvid != "" {
			link += v.Bvid
		} else {
			link += "av" + strconv.FormatInt(v.Aid, 10)
		}
		desc := renderBilibiliUGCDescription(embed, v.Pic, html.EscapeString(v.Description), v.Aid, 0, v.Bvid)
		desc = strings.TrimSpace(desc + "<br/>Length: " + html.EscapeString(v.Length))
		item := routeutils.NewItem(v.Title, link, desc, time.Unix(v.Created, 0))
		if item == nil {
			return nil
		}
		item.GUID = "bilibili-user-video-" + strconv.FormatInt(v.Aid, 10)
		if v.Author != "" {
			routeutils.SetAuthor(item, v.Author, routeutils.WithAuthorURI(spaceURL(strconv.FormatInt(v.Mid, 10))))
		}
		return item
	})
}

// --- user/dynamic: UP 主动态 ---

type bilibiliDynamicItem struct {
	IDStr   string `json:"id_str"`
	Modules struct {
		ModuleAuthor struct {
			Mid    int64  `json:"mid"`
			Name   string `json:"name"`
			Face   string `json:"face"`
			PubTs  int64  `json:"pub_ts"`
			Action int64  `json:"action"`
		} `json:"module_author"`
		ModuleDynamic struct {
			Desc struct {
				Text string `json:"text"`
			} `json:"desc"`
			Major *bilibiliDynamicMajor `json:"major"`
		} `json:"module_dynamic"`
	} `json:"modules"`
}

type bilibiliDynamicMajor struct {
	Type string `json:"type"`
	None *struct {
		Tips string `json:"tips"`
	} `json:"none"`
	Archive *struct {
		Aid          jsonInt64 `json:"aid"`
		Bvid         string    `json:"bvid"`
		Title        string    `json:"title"`
		Desc         string    `json:"desc"`
		Pic          string    `json:"pic"`
		DurationText string    `json:"duration_text"`
	} `json:"archive"`
	Opus *struct {
		Title   string `json:"title"`
		Summary *struct {
			Text string `json:"text"`
		} `json:"summary"`
		Pics []struct {
			URL    string `json:"url"`
			Width  int64  `json:"width"`
			Height int64  `json:"height"`
		} `json:"pics"`
	} `json:"opus"`
	Draw *struct {
		Items []struct {
			Src    string `json:"src"`
			Width  int64  `json:"width"`
			Height int64  `json:"height"`
		} `json:"items"`
	} `json:"draw"`
	Article *struct {
		ID     int64    `json:"id"`
		Title  string   `json:"title"`
		Covers []string `json:"covers"`
	} `json:"article"`
	Common *struct {
		Title   string `json:"title"`
		Desc    string `json:"desc"`
		JumpURL string `json:"jump_url"`
	} `json:"common"`
	Live *struct {
		DescFirst  string `json:"desc_first"`
		DescSecond string `json:"desc_second"`
	} `json:"live"`
	LiveRcmd *struct {
		Content string `json:"content"` // embedded JSON string
	} `json:"live_rcmd"`
}

type bilibiliLivePlayInfo struct {
	LivePlayInfo struct {
		Title       string `json:"title"`
		Cover       string `json:"cover"`
		AreaName    string `json:"area_name"`
		RoomID      int64  `json:"room_id"`
		WatchedShow struct {
			TextLarge string `json:"text_large"`
		} `json:"watched_show"`
	} `json:"live_play_info"`
}

type bilibiliDynamicFeedResp struct {
	biliResp
	Data struct {
		HasMore bool                  `json:"has_more"`
		Items   []bilibiliDynamicItem `json:"items"`
	} `json:"data"`
}

// BilibiliUserDynamicHandler handles /bilibili/user/dynamic/:uid/:routeParams?
// routeParams is an RSSHub-style query fragment, e.g. "embed=0&showEmoji=1".
func BilibiliUserDynamicHandler(c *ctxpkg.Context) (*models.Feed, error) {
	uid := c.Param("uid")
	if uid == "" {
		return nil, fmt.Errorf("bilibili: missing uid")
	}
	opts := parseDynamicParams(c.Param("routeParams"))
	ctx := c.Parent()

	apiURL := "https://api.bilibili.com/x/polymer/web-dynamic/v1/feed/space?host_mid=" + url.QueryEscape(uid) + "&offset="
	// Anonymous calls get banned by the gaia gateway (-412/-352); a buvid
	// pair from finger/spi is enough to pass without login.
	profile := disguiseProfileForSpace(uid).Referer(spaceURL(uid) + "/dynamic")
	if cookie, cerr := bilibiliBuvidCookie(c); cerr == nil {
		profile = profile.Cookie(cookie)
	}
	var resp bilibiliDynamicFeedResp
	if err := profile.Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(fmt.Sprintf("%s 的 bilibili 动态", uid), spaceURL(uid)+"/dynamic", "bilibili 用户动态")
	appendUserDynamics(feed, resp.Data.Items, opts)
	return feed, nil
}

type bilibiliDynamicOpts struct {
	Embed bool
}

// parseDynamicParams parses the optional "k=v&k2=v2" style path fragment.
func parseDynamicParams(raw string) bilibiliDynamicOpts {
	opts := bilibiliDynamicOpts{Embed: true}
	for _, kv := range strings.Split(raw, "&") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(parts[0])) {
		case "embed":
			opts.Embed = routeutils.ParseBool(parts[1], true)
		}
	}
	return opts
}

// appendUserDynamics maps polymer dynamic entries into feed items.
func appendUserDynamics(feed *models.Feed, items []bilibiliDynamicItem, opts bilibiliDynamicOpts) {
	routeutils.AppendMappedItems(feed, items, 0, func(d bilibiliDynamicItem) *models.Item {
		major := d.Modules.ModuleDynamic.Major
		title, link := bilibiliDynamicTitleLink(d.IDStr, d.Modules.ModuleAuthor.Mid, major)
		if title == "" {
			title = collapseDynamicText(d.Modules.ModuleDynamic.Desc.Text)
		}
		if title == "" {
			return nil
		}
		if link == "" {
			link = "https://t.bilibili.com/" + d.IDStr
		}

		imgs := majorImages(major)
		cover := ""
		if len(imgs) > 0 {
			cover = imgs[0]
		}
		var b strings.Builder
		b.WriteString(renderBilibiliUGCDescription(opts.Embed && major != nil && major.Archive != nil,
			cover, "", dynAid(major), 0, dynBvid(major)))
		if text := collapseDynamicText(d.Modules.ModuleDynamic.Desc.Text); text != "" {
			b.WriteString(html.EscapeString(text))
			b.WriteString("<br/>")
		}
		if major != nil && major.Archive != nil && major.Archive.Desc != "" {
			b.WriteString(html.EscapeString(major.Archive.Desc))
			b.WriteString("<br/>")
		}
		for _, img := range imgs[1:] { // additional pictures beyond the cover
			b.WriteString(fmt.Sprintf(`<img src="%s"/><br/>`, html.EscapeString(img)))
		}

		item := routeutils.NewItem(title, link, b.String(), time.Unix(d.Modules.ModuleAuthor.PubTs, 0))
		if item == nil {
			return nil
		}
		item.GUID = "bilibili-user-dynamic-" + d.IDStr
		if name := d.Modules.ModuleAuthor.Name; name != "" {
			routeutils.SetAuthor(item, name, routeutils.WithAuthorURI(spaceURL(strconv.FormatInt(d.Modules.ModuleAuthor.Mid, 10))))
		}
		return item
	})
}

// bilibiliDynamicTitleLink derives title and link from the major block.
func bilibiliDynamicTitleLink(idStr string, mid int64, m *bilibiliDynamicMajor) (string, string) {
	switch {
	case m == nil:
		return "", ""
	case m.None != nil:
		return m.None.Tips, ""
	case m.Archive != nil:
		link := bilibiliBaseURL + "/video/"
		if m.Archive.Bvid != "" {
			link += m.Archive.Bvid
		} else if m.Archive.Aid.Int64() > 0 {
			link += "av" + strconv.FormatInt(m.Archive.Aid.Int64(), 10)
		} else {
			link = ""
		}
		return m.Archive.Title, link
	case m.Opus != nil:
		return firstNonEmpty(m.Opus.Title, collapseDynamicText(opusSummaryText(m))), ""
	case m.Draw != nil:
		return "", ""
	case m.Article != nil:
		return m.Article.Title, fmt.Sprintf("%s/read/cv%d", bilibiliBaseURL, m.Article.ID)
	case m.Common != nil:
		return firstNonEmpty(m.Common.Title, m.Common.Desc), m.Common.JumpURL
	case m.Live != nil:
		return strings.TrimSpace(m.Live.DescFirst + "<br/>" + m.Live.DescSecond), ""
	case m.LiveRcmd != nil:
		var info bilibiliLivePlayInfo
		if err := unmarshalJSONString(m.LiveRcmd.Content, &info); err == nil {
			lpi := info.LivePlayInfo
			title := strings.TrimSpace(lpi.AreaName + "·" + lpi.WatchedShow.TextLarge)
			if lpi.RoomID != 0 {
				return firstNonEmpty(lpi.Title, title), liveRoomURL(lpi.RoomID)
			}
			return title, ""
		}
		return "", ""
	default:
		return "", ""
	}
}

func opusSummaryText(m *bilibiliDynamicMajor) string {
	if m == nil || m.Opus == nil || m.Opus.Summary == nil {
		return ""
	}
	return m.Opus.Summary.Text
}

// majorImages collects cover/photo URLs from the major block.
func majorImages(m *bilibiliDynamicMajor) []string {
	if m == nil {
		return nil
	}
	var out []string
	add := func(u string) {
		if u != "" {
			out = append(out, u)
		}
	}
	switch {
	case m.Archive != nil:
		add(m.Archive.Pic)
	case m.Opus != nil:
		for _, p := range m.Opus.Pics {
			add(p.URL)
		}
	case m.Draw != nil:
		for _, p := range m.Draw.Items {
			add(p.Src)
		}
	case m.Article != nil:
		for _, c := range m.Article.Covers {
			add(c)
		}
	case m.LiveRcmd != nil:
		var info bilibiliLivePlayInfo
		if err := unmarshalJSONString(m.LiveRcmd.Content, &info); err == nil {
			add(info.LivePlayInfo.Cover)
		}
	}
	return out
}

func dynAid(m *bilibiliDynamicMajor) int64 {
	if m != nil && m.Archive != nil {
		return m.Archive.Aid.Int64()
	}
	return 0
}

func dynBvid(m *bilibiliDynamicMajor) string {
	if m != nil && m.Archive != nil {
		return m.Archive.Bvid
	}
	return ""
}

func collapseDynamicText(s string) string {
	return strings.TrimSpace(s)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
