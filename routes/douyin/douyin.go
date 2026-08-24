package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var douyinHotSearchRoute = routeutils.RouteSpec{
	Path:        "hot-search",
	Name:        "Hot Search",
	Example:     "douyin/hot-search",
	Maintainers: []string{"xihale"},
	Description: "Douyin trending hot search keywords with covers and heat values",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Maximum number of items, default 50, max 50"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  DouyinHotSearchHandler,
}

// DouyinHotSearchHandler handles /douyin/hot-search
func DouyinHotSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 50, 50)
	ctx := c.Parent()

	var resp douyinHotResp
	if err := disguise.Chrome().Lang("zh-CN,zh;q=0.9").JSONAccept().
		Referer("https://www.douyin.com/").
		Fetch("https://www.douyin.com/aweme/v1/web/hot/search/list/").
		GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.StatusCode != 0 && resp.StatusCode != 2052 {
		return nil, fmt.Errorf("douyin api error %d", resp.StatusCode)
	}

	feed := routeutils.NewFeed("抖音热搜榜", "https://www.douyin.com/hot", "抖音实时热搜榜")
	for _, entry := range resp.Data.WordList {
		if entry.Word == "" {
			continue
		}
		desc := ""
		if cover := entry.LastURL(); cover != "" {
			desc += fmt.Sprintf(`<img src="%s" alt=""/><br/>`, html.EscapeString(cover))
		}
		desc += html.EscapeString(entry.Word) + fmt.Sprintf("<br/>热度: %d | 视频数: %d", entry.HotValue, entry.VideoCount)
		link := "https://www.douyin.com/search/" + url.PathEscape(entry.Word)

		var pubDate time.Time
		if entry.EventTime > 0 {
			pubDate = time.Unix(entry.EventTime, 0)
		}
		item := routeutils.NewItem(entry.Word, link, desc, pubDate)
		if item == nil {
			continue
		}
		guid := entry.GroupID
		if guid == "" {
			guid = entry.SentenceID
		}
		if guid != "" {
			item.GUID = "douyin-hot-" + guid
		}
		routeutils.AddItem(feed, item)
		if limit > 0 && len(feed.Items) >= limit {
			break
		}
	}
	return feed, nil
}

type douyinHotResp struct {
	StatusCode int `json:"status_code"`
	Data       struct {
		ActiveTime json.RawMessage     `json:"active_time"`
		WordList   []douyinHotWordItem `json:"word_list"`
	} `json:"data"`
}

type douyinHotWordItem struct {
	Word       string `json:"word"`
	HotValue   int64  `json:"hot_value"`
	EventTime  int64  `json:"event_time"`
	GroupID    string `json:"group_id"`
	SentenceID string `json:"sentence_id"`
	VideoCount int    `json:"video_count"`
	WordCover  struct {
		URLList []string `json:"url_list"`
	} `json:"word_cover"`
}

// LastURL returns the last (usually highest resolution) cover URL.
func (w douyinHotWordItem) LastURL() string {
	n := len(w.WordCover.URLList)
	if n == 0 {
		return ""
	}
	return w.WordCover.URLList[n-1]
}
