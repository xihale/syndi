package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// baiduProfile disguises requests against top.baidu.com.
var baiduProfile = disguise.Chrome().Lang("zh-CN,zh;q=0.9")

type baiduTopCardItem struct {
	Word   string   `json:"word"`
	Desc   string   `json:"desc"`
	RawURL string   `json:"rawUrl"`
	URL    string   `json:"url"`
	Img    string   `json:"img"`
	Show   []string `json:"show"`
}

type baiduTopPayload struct {
	Data struct {
		CurBoardName string `json:"curBoardName"`
		Cards        []struct {
			Content []baiduTopCardItem `json:"content"`
		} `json:"cards"`
	} `json:"data"`
}

var baiduTopRoute = routeutils.RouteSpec{
	Path:        "top",
	Name:        "Baidu Hot Search",
	Example:     "baidu/top",
	Maintainers: []string{"xihale"},
	Description: "Baidu (百度) real-time hot search rankings",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     BaiduTopHandler,
}

var baiduTopBoardRoute = routeutils.RouteSpec{
	Path:        "top/:board",
	Name:        "Baidu Hot Search Board",
	Example:     "baidu/top/novel",
	Maintainers: []string{"xihale"},
	Description: "Baidu (百度) hot rankings by board",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("board", "realtime 热搜榜, novel 小说榜, movie 电影榜, teleplay 电视剧榜, car 汽车榜 or game 游戏榜"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  BaiduTopHandler,
}

var baiduBoards = map[string]string{
	"realtime": "热搜榜", "novel": "小说榜", "movie": "电影榜",
	"teleplay": "电视剧榜", "car": "汽车榜", "game": "游戏榜",
}

// BaiduTopHandler handles /baidu/top/:board?
func BaiduTopHandler(c *ctxpkg.Context) (*models.Feed, error) {
	board := c.Param("board")
	if board == "" {
		board = "realtime"
	}
	if _, ok := baiduBoards[board]; !ok {
		return nil, fmt.Errorf("baidu: unknown board %q", board)
	}
	pageURL := fmt.Sprintf("https://top.baidu.com/board?tab=%s", board)

	pageHTML, err := baiduProfile.Fetch(pageURL).GetString(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}

	payload, err := parseBaiduSData(pageHTML)
	if err != nil {
		return nil, err
	}

	title := payload.Data.CurBoardName
	if title == "" {
		title = baiduBoards[board]
	}
	feed := routeutils.NewFeed(title+" - 百度热搜", pageURL, "百度"+title)

	for _, card := range payload.Data.Cards {
		for _, it := range card.Content {
			word := strings.TrimSpace(it.Word)
			if word == "" {
				continue
			}
			link := it.RawURL
			if link == "" {
				link = it.URL
			}
			var b strings.Builder
			if it.Img != "" {
				b.WriteString(`<img src="` + html.EscapeString(it.Img) + `"/><br>`)
			}
			for _, s := range it.Show {
				if s = strings.TrimSpace(s); s != "" {
					b.WriteString(html.EscapeString(s) + "<br>")
				}
			}
			if d := strings.TrimSpace(it.Desc); d != "" {
				b.WriteString("<p>" + html.EscapeString(d) + "</p>")
			}
			item := routeutils.NewItem(word, link, b.String(), time.Time{})
			item.GUID = "baidu:top:" + word
			routeutils.AddItem(feed, item)
		}
	}
	return feed, nil
}

// parseBaiduSData extracts the SSR JSON blob embedded after "s-data:" inside
// an HTML comment of the page.
func parseBaiduSData(pageHTML string) (*baiduTopPayload, error) {
	idx := strings.Index(pageHTML, "s-data:")
	if idx < 0 {
		return nil, fmt.Errorf("baidu: s-data payload not found")
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(pageHTML[idx+len("s-data:"):])))
	var payload baiduTopPayload
	if err := dec.Decode(&payload); err != nil {
		return nil, fmt.Errorf("baidu: bad s-data JSON: %w", err)
	}
	return &payload, nil
}
