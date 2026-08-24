package routes

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// ttUserAgent must stay in sync with the UA fed into the a_bogus signature.
const ttUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// ttProfile pins the exact UA so signature and request always match.
var ttProfile = disguise.Custom(ttUserAgent).JSONAccept().Referer("https://www.toutiao.com/")

// --- Upstream payload types ---

type ttResp struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type ttUserInfo struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Desc      string `json:"desc"`
}

// ttUserCell covers both {info:{...}} wrapped and flat user shapes.
type ttUserCell struct {
	Info      *ttUserInfo `json:"info"`
	Name      string      `json:"name"`
	AvatarURL string      `json:"avatar_url"`
	Desc      string      `json:"desc"`

	Description string `json:"description"` // user_info style
}

func (u *ttUserCell) displayName() string {
	if u == nil {
		return ""
	}
	if u.Info != nil && u.Info.Name != "" {
		return u.Info.Name
	}
	return u.Name
}

func (u *ttUserCell) avatar() string {
	if u == nil {
		return ""
	}
	if u.Info != nil && u.Info.AvatarURL != "" {
		return u.Info.AvatarURL
	}
	return u.AvatarURL
}

func (u *ttUserCell) bio() string {
	if u == nil {
		return ""
	}
	if u.Info != nil && u.Info.Desc != "" {
		return u.Info.Desc
	}
	if u.Desc != "" {
		return u.Desc
	}
	return u.Description
}

type ttPlayAddrListItem struct {
	Bitrate     int      `json:"bitrate"`
	PlayURLList []string `json:"play_url_list"`
}

type ttVideo struct {
	OriginCover struct {
		URLList []string `json:"url_list"`
	} `json:"origin_cover"`
	PlayAddrList []ttPlayAddrListItem `json:"play_addr_list"`
}

// bestPlayURL returns the highest-bitrate playback URL.
func (v *ttVideo) bestPlayURL() string {
	if v == nil || len(v.PlayAddrList) == 0 {
		return ""
	}
	sorted := make([]ttPlayAddrListItem, len(v.PlayAddrList))
	copy(sorted, v.PlayAddrList)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Bitrate > sorted[j].Bitrate })
	for _, p := range sorted[0].PlayURLList {
		if p != "" {
			return p
		}
	}
	return ""
}

func (v *ttVideo) coverURL() string {
	if v == nil || len(v.OriginCover.URLList) == 0 {
		return ""
	}
	return v.OriginCover.URLList[0]
}

type ttFeed struct {
	ID          string      `json:"id"`
	CellType    int         `json:"cell_type"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	RichContent string      `json:"rich_content"`
	Abstract    string      `json:"abstract"`
	PublishTime int64       `json:"publish_time"` // unix seconds
	Source      string      `json:"source"`
	User        *ttUserCell `json:"user"`
	UserInfo    *ttUserCell `json:"user_info"`
	Video       *ttVideo    `json:"video"`
}

func (f *ttFeed) author() string {
	if name := f.User.displayName(); name != "" {
		return name
	}
	if f.UserInfo != nil {
		return f.UserInfo.displayName()
	}
	return f.Source
}

var toutiaoUserRoute = routeutils.RouteSpec{
	Path:        "user/token/:token",
	Name:        "Toutiao User Profile",
	Example:     "toutiao/user/token/MS4wLjABAAAA_Q07NxeCa4hDPFoRcdphaZOk2X6C8BApfpTPTMLJswI",
	Maintainers: []string{"xihale"},
	Description: "Posts of a Toutiao (今日头条) user profile",
	Categories:  []models.Category{{Name: "new-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("token", "user token from the profile page URL"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  ToutiaoUserHandler,
}

// ToutiaoUserHandler handles /toutiao/user/token/:token
func ToutiaoUserHandler(c *ctxpkg.Context) (*models.Feed, error) {
	token := c.Param("token")
	query := fmt.Sprintf("category=profile_all&token=%s&max_behot_time=0&entrance_gid&aid=24&app_name=toutiao_web", token)
	signature := generateABogus(query, ttUserAgent)
	apiURL := "https://www.toutiao.com/api/pc/list/feed?" + query + "&a_bogus=" + signature

	var resp ttResp
	if err := ttProfile.Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	var feedItems []ttFeed
	if err := json.Unmarshal(resp.Data, &feedItems); err != nil {
		return nil, fmt.Errorf("toutiao: unexpected payload (signature likely rejected): %w", err)
	}
	if len(feedItems) == 0 {
		return nil, fmt.Errorf("toutiao: no posts returned for token %q", token)
	}

	feed := routeutils.NewFeedWithOptions(routeutils.FeedOptions{
		Title:       fmt.Sprintf("%s的头条主页 - 今日头条(www.toutiao.com)", feedItems[0].author()),
		Link:        "https://www.toutiao.com/c/user/token/" + token + "/",
		Description: feedItems[0].User.bio(),
		Image:       feedItems[0].User.avatar(),
	})

	for _, it := range feedItems {
		pubDate := time.Time{}
		if it.PublishTime > 0 {
			pubDate = time.Unix(it.PublishTime, 0)
		}
		var title, desc, link string
		switch it.CellType {
		case 0, 49: // video
			title = strings.TrimSpace(it.Title)
			play := it.Video.bestPlayURL()
			cover := it.Video.coverURL()
			if play != "" {
				desc = fmt.Sprintf(`<video controls preload="metadata" poster="%s"><source src="%s" type="video/mp4"/></video>`, cover, play)
			}
			link = "https://www.toutiao.com/video/" + it.ID + "/"
		case 32: // text post without title
			desc = it.RichContent
			for _, line := range strings.Split(it.Content, "\n") {
				if line = strings.TrimSpace(line); line != "" {
					title = line
					break
				}
			}
			link = "https://www.toutiao.com/w/" + it.ID + "/"
		default: // 60+ articles
			title = strings.TrimSpace(it.Title)
			desc = it.Abstract
			link = "https://www.toutiao.com/article/" + it.ID + "/"
		}
		if title == "" || link == "" {
			continue
		}
		item := routeutils.NewItem(title, link, desc, pubDate)
		item.GUID = it.ID
		routeutils.SetItemAuthor(item, it.author(), "", "")
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
