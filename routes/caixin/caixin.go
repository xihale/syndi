package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

// caixinProfile disguises requests against caixin.com endpoints.
var caixinProfile = disguise.Chrome().Lang("zh-CN,zh;q=0.9")

var caixinColumns = []string{"economy", "finance", "china", "science", "international", "opinion", "culture", "weekly"}

// --- Upstream payloads ---

type caixinScrollResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ArticleList []caixinScrollItem `json:"articleList"`
	} `json:"data"`
}

type caixinScrollItem struct {
	Title         string         `json:"title"`
	Summary       string         `json:"summary"`
	URL           string         `json:"url"`
	Time          int64          `json:"time"` // unix milliseconds
	ChannelObject *caixinChannel `json:"channelObject"`
}

type caixinChannel struct {
	Name string `json:"name"`
}

type caixinHomeResp struct {
	Datas []caixinHomeItem `json:"datas"`
}

type caixinHomeItem struct {
	Desc     string `json:"desc"`
	Summ     string `json:"summ"`
	Link     string `json:"link"`
	Time     string `json:"time"` // "2020-04-10 12:53:44" Beijing time
	Keyword  string `json:"keyword"`
	AudioURL string `json:"audioUrl"`
}

func caixinEscapeText(s string) string {
	return strings.TrimSpace(html.EscapeString(s))
}

var caixinLatestRoute = routeutils.RouteSpec{
	Path:        "latest",
	Name:        "Caixin Latest",
	Example:     "caixin/latest",
	Maintainers: []string{"xihale"},
	Description: "Caixin (财新网) latest articles",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     CaixinLatestHandler,
}

// CaixinLatestHandler handles /caixin/latest
func CaixinLatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var resp caixinScrollResp
	if err := caixinProfile.Fetch("https://gateway.caixin.com/api/dataplatform/scroll/index").
		GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("caixin api error %d: %s", resp.Code, resp.Msg)
	}

	feed := routeutils.NewFeed("财新网 - 最新文章", "https://www.caixin.com/", "财新网最新文章")
	for _, e := range resp.Data.ArticleList {
		link := e.URL
		if link == "" || e.Title == "" ||
			strings.HasPrefix(link, "https://fm.caixin.com/") ||
			strings.HasPrefix(link, "https://video.caixin.com/") ||
			strings.HasPrefix(link, "https://datanews.caixin.com/") {
			continue
		}
		desc := ""
		if s := caixinEscapeText(e.Summary); s != "" {
			desc = "<p>" + s + "</p>"
		}
		pubDate := time.Time{}
		if e.Time > 0 {
			pubDate = time.UnixMilli(e.Time)
		}
		item := routeutils.NewItem(e.Title, link, desc, pubDate)
		item.GUID = "caixin:latest:" + link
		if e.ChannelObject != nil {
			routeutils.SetCategories(item, e.ChannelObject.Name)
		}
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

var _regexpEntityJSON = regexp.MustCompile(`(?s)var entity = (\{.*?\})`)

var caixinCategoryRoute = routeutils.RouteSpec{
	Path:        "category/:column/:category",
	Name:        "Caixin Column Category",
	Example:     "caixin/category/finance/regulation",
	Maintainers: []string{"xihale"},
	Description: "Caixin (财新网) articles by column and category",
	Categories:  []models.Category{{Name: "new-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("column", "economy, finance, china, science, international, opinion, culture or weekly"),
		routeutils.RequiredParam("category", "sub-category slug, e.g. regulation, bank, stock"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  CaixinCategoryHandler,
}

// CaixinCategoryHandler handles /caixin/category/:column/:category
func CaixinCategoryHandler(c *ctxpkg.Context) (*models.Feed, error) {
	column := c.Param("column")
	category := c.Param("category")
	valid := false
	for _, col := range caixinColumns {
		if col == column {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("caixin: invalid column %q", column)
	}
	pageURL := fmt.Sprintf("https://%s.caixin.com/%s", column, category)

	doc, err := caixinProfile.Fetch(pageURL).GetHTML(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}
	m := _regexpEntityJSON.FindStringSubmatch(doc.Find("script").Text())
	if m == nil {
		return nil, fmt.Errorf("caixin: entity JSON not found on %s", pageURL)
	}
	var entity struct {
		ID    int64  `json:"id"`
		Cdesc string `json:"cdesc"`
	}
	if err := json.Unmarshal([]byte(m[1]), &entity); err != nil {
		return nil, fmt.Errorf("caixin: bad entity JSON on %s: %w", pageURL, err)
	}

	apiURL := fmt.Sprintf("https://gateway.caixin.com/api/extapi/homeInterface.jsp?subject=%d&type=0&count=25&picdim=_266_177&start=0", entity.ID)
	var home caixinHomeResp
	if err := caixinProfile.Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &home); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(doc.Text("head title"))
	feed := routeutils.NewFeed(title, pageURL, "财新网 - 提供财经新闻及资讯服务")
	for _, it := range home.Datas {
		link := strings.Replace(it.Link, "http://", "https://", 1)
		titleText := strings.TrimSpace(it.Desc)
		if link == "" || titleText == "" {
			continue
		}
		desc := ""
		if s := caixinEscapeText(it.Summ); s != "" {
			desc = "<p>" + s + "</p>"
		}
		pubDate := time.Time{}
		if it.Time != "" {
			if t, perr := dateutil.ParseDateInLocation(it.Time, time.FixedZone("UTC+8", 8*3600)); perr == nil {
				pubDate = t
			}
		}
		item := routeutils.NewItem(titleText, link, desc, pubDate)
		item.GUID = "caixin:category:" + link
		routeutils.SetCategories(item, strings.Split(it.Keyword, " ")...)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
