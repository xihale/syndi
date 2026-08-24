package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// juejinPinTitle maps pins type values to feed titles.
var juejinPinTitle = map[string]string{
	"recommend":           "推荐",
	"hot":                 "热门",
	"6824710203301167112": "上班摸鱼",
	"6819970850532360206": "内推招聘",
	"6824710202487472141": "一图胜千言",
	"6824710202562969614": "今天学到了",
	"6824710202378436621": "每天一道算法题",
	"6824710202000932877": "开发工具推荐",
	"6824710203112423437": "树洞一下",
}

type juejinPinMsgInfo struct {
	MsgID   string   `json:"msg_id"`
	Content string   `json:"content"`
	Ctime   string   `json:"ctime"` // unix seconds
	PicList []string `json:"pic_list"`
}

type juejinPin struct {
	MsgID          string           `json:"msg_id"`
	MsgInfo        juejinPinMsgInfo `json:"msg_Info"`
	AuthorUserInfo juejinAuthorInfo `json:"author_user_info"`
}

var juejinPinsRoute = routeutils.RouteSpec{
	Path:        "pins",
	Name:        "Juejin Pins",
	Example:     "juejin/pins",
	Maintainers: []string{"xihale"},
	Description: "Juejin Pins (沸点) recommended short posts",
	Categories:  []models.Category{{Name: "programming"}},
	CacheTTL:    30 * time.Minute,
	Handler:     JuejinPinsHandler,
}

var juejinPinsTypeRoute = routeutils.RouteSpec{
	Path:        "pins/:type",
	Name:        "Juejin Pins By Type",
	Example:     "juejin/pins/hot",
	Maintainers: []string{"xihale"},
	Description: "Juejin Pins (沸点) by tab or numeric topic id",
	Categories:  []models.Category{{Name: "programming"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("type", "hot, 上班摸鱼=6824710203301167112, 内推招聘=6819970850532360206, or another topic id"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  JuejinPinsHandler,
}

// JuejinPinsHandler handles /juejin/pins/:type?
func JuejinPinsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	typ := c.Param("type")
	if typ == "" {
		typ = "recommend"
	}
	title := "沸点"
	if t, ok := juejinPinTitle[typ]; ok {
		title = fmt.Sprintf("沸点 - %s", t)
	}

	endpoint := ""
	body := map[string]any{}
	if isJuejinNumeric(typ) {
		endpoint = "https://api.juejin.cn/recommend_api/v1/short_msg/topic"
		body = map[string]any{"id_type": 4, "sort_type": 500, "cursor": "0", "limit": 20, "topic_id": typ}
	} else {
		if _, ok := juejinPinTitle[typ]; !ok {
			return nil, fmt.Errorf("juejin: unknown pins type %q", typ)
		}
		endpoint = "https://api.juejin.cn/recommend_api/v1/short_msg/" + typ
		body = map[string]any{"id_type": 4, "sort_type": 200, "cursor": "0", "limit": 20}
	}

	var resp juejinResp
	if err := juejinProfile.PostJSON(endpoint, body).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.ok(); err != nil {
		return nil, err
	}
	var pins []juejinPin
	if err := json.Unmarshal(resp.Data, &pins); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(title, "https://juejin.cn/pins/recommended", title)
	for _, p := range pins {
		content := strings.TrimSpace(p.MsgInfo.Content)
		if content == "" || p.MsgInfo.MsgID == "" {
			continue
		}
		var b strings.Builder
		b.WriteString("<p>" + strings.ReplaceAll(html.EscapeString(content), "\n", "<br>") + "</p>")
		for _, img := range p.MsgInfo.PicList {
			b.WriteString(`<img src="` + html.EscapeString(img) + `"/><br>`)
		}
		pubDate := time.Time{}
		if sec, err := strconv.ParseInt(p.MsgInfo.Ctime, 10, 64); err == nil && sec > 0 {
			pubDate = time.Unix(sec, 0)
		}
		item := routeutils.NewItem(content, "https://juejin.cn/pin/"+p.MsgInfo.MsgID, b.String(), pubDate)
		item.GUID = p.MsgInfo.MsgID
		routeutils.SetItemAuthor(item, p.AuthorUserInfo.UserName, "", "")
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

func isJuejinNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}
