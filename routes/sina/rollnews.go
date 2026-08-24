package routes

import (
	"fmt"
	"html"
	"math/rand"
	"strconv"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const sinaRollPageID = "153"

// sinaRollLids maps rolling news section ids to display names.
var sinaRollLids = map[string]string{
	"2509": "全部", "2510": "国内", "2511": "国际", "2669": "社会",
	"2512": "体育", "2513": "娱乐", "2514": "军事", "2515": "科技",
	"2516": "财经", "2517": "股市", "2518": "美股",
}

func sinaProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer("https://news.sina.com.cn/")
}

type sinaRollItem struct {
	Title     string `json:"title"`
	Intro     string `json:"intro"`
	URL       string `json:"url"`
	WapURL    string `json:"wapurl"`
	InTime    string `json:"intime"`
	MTime     string `json:"mtime"`
	MediaName string `json:"media_name"`
	DocID     string `json:"docid"`
}

type sinaRollResp struct {
	Result struct {
		Data []sinaRollItem `json:"data"`
	} `json:"result"`
}

var sinaRollNewsRoute = routeutils.RouteSpec{
	Path:        "rollnews",
	Name:        "Sina Rolling News",
	Example:     "sina/rollnews",
	Maintainers: []string{"xihale"},
	Description: "Sina (新浪) rolling news, all sections",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Max items, default 30"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  SinaRollNewsHandler,
}

var sinaRollNewsSectionRoute = routeutils.RouteSpec{
	Path:        "rollnews/:lid",
	Name:        "Sina Rolling News Section",
	Example:     "sina/rollnews/2669",
	Maintainers: []string{"xihale"},
	Description: "Sina (新浪) rolling news by section. Lids: 2509 全部, 2510 国内, 2511 国际, 2669 社会, 2512 体育, 2513 娱乐, 2514 军事, 2515 科技, 2516 财经, 2517 股市, 2518 美股",
	Categories:  []models.Category{{Name: "new-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("lid", "Section id, e.g. 2515 for tech"),
		routeutils.OptionalParam("limit", "Max items, default 30"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  SinaRollNewsHandler,
}

// SinaRollNewsHandler handles /sina/rollnews and /sina/rollnews/:lid
func SinaRollNewsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	lid := c.Param("lid")
	if lid == "" {
		lid = "2509"
	}
	if _, ok := sinaRollLids[lid]; !ok {
		if _, err := strconv.Atoi(lid); err != nil {
			return nil, fmt.Errorf("invalid lid %q; see route description", lid)
		}
	}
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 50)

	apiURL := fmt.Sprintf(
		"https://feed.mix.sina.com.cn/api/roll/get?pageid=%s&lid=%s&k=&num=%d&page=1&r=%f&_=%d",
		sinaRollPageID, lid, limit, rand.Float64(), time.Now().UnixMilli(),
	)
	var resp sinaRollResp
	if err := sinaProfile().JSONAccept().Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}

	name := lid
	if n, ok := sinaRollLids[lid]; ok {
		name = n
	}
	feed := routeutils.NewFeed(
		"新浪"+name+"滚动新闻",
		fmt.Sprintf("https://news.sina.com.cn/roll/#pageid=%s&lid=%s", sinaRollPageID, lid),
		"新浪"+name+"滚动新闻",
	)

	for _, r := range resp.Result.Data {
		if r.Title == "" || r.URL == "" {
			continue
		}
		desc := ""
		if r.Intro != "" {
			desc = "<p>" + html.EscapeString(r.Intro) + "</p>"
		}
		item := routeutils.NewItem(r.Title, r.URL, desc, sinaParseUnix(r.InTime))
		if item == nil {
			continue
		}
		if r.DocID != "" {
			item.GUID = r.DocID
		}
		if mt := sinaParseUnix(r.MTime); !mt.IsZero() {
			routeutils.SetUpdated(item, mt)
		}
		if r.MediaName != "" {
			routeutils.SetItemAuthor(item, r.MediaName, "", "")
		}
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

func sinaParseUnix(s string) time.Time {
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil || sec <= 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).In(time.FixedZone("CST", 8*60*60))
}
