package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

// jin10Profile disguises requests against the jin10 APIs. The x-app-id /
// x-version headers are the public values shipped by upstream web clients.
var jin10Profile = disguise.Chrome().JSONAccept().
	WithHeader("x-app-id", "bVBF4FyRTn5NJF5n").
	WithHeader("x-version", "1.0.0")

// jin10Z3cProfile targets the z3c classified-flash host, which rejects
// x-version 1.0.0 with 502 and requires the older 1.0 value.
var jin10Z3cProfile = disguise.Chrome().JSONAccept().
	WithHeader("x-app-id", "bVBF4FyRTn5NJF5n").
	WithHeader("x-version", "1.0")

type jin10Resp struct {
	Status int             `json:"status"`
	Data   json.RawMessage `json:"data"`
}

type jin10FlashData struct {
	Content  string `json:"content"`
	Title    string `json:"title"`
	Pic      string `json:"pic"`
	Link     string `json:"link"`
	VipTitle string `json:"vip_title"`
}

type jin10Flash struct {
	ID        string         `json:"id"`
	Type      int            `json:"type"`
	Important int            `json:"important"`
	Time      string         `json:"time"` // "2026-08-24 17:37:46" Beijing time
	Data      jin10FlashData `json:"data"`
}

// jin10CST is China Standard Time for naive upstream timestamps.
var jin10CST = time.FixedZone("UTC+8", 8*3600)

// mapJin10Flashes converts flash entries into feed items.
func mapJin10Flashes(feed *models.Feed, flashes []jin10Flash, guidPrefix string) {
	for _, f := range flashes {
		if f.Type == 1 { // ads
			continue
		}
		content := f.Data.Content
		title := ""
		// 【标题】 prefix carries the headline; strip it from the body.
		if rest := strings.TrimPrefix(content, "【"); rest != content {
			if end := strings.Index(rest, "】"); end >= 0 {
				title = rest[:end]
				content = strings.TrimSpace(rest[end+len("】"):])
			}
		}
		if title == "" {
			title = strings.TrimSpace(f.Data.VipTitle)
		}
		if title == "" {
			title = strings.TrimSpace(content)
		}
		if title == "" {
			continue
		}
		desc := "<p>" + strings.ReplaceAll(html.EscapeString(content), "\n", "<br>") + "</p>"
		if pic := f.Data.Pic; pic != "" {
			desc += `<img src="` + html.EscapeString(pic) + `"/>`
		}
		pubDate := time.Time{}
		if f.Time != "" {
			if t, err := dateutil.ParseDateInLocation(f.Time, jin10CST); err == nil {
				pubDate = t
			}
		}
		link := f.Data.Link
		item := routeutils.NewItem(title, link, desc, pubDate)
		item.GUID = fmt.Sprintf("%s:%s", guidPrefix, f.ID)
		routeutils.AddItem(feed, item)
	}
}

var jin10FlashRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Jin10 Market News",
	Example:     "jin10",
	Maintainers: []string{"xihale"},
	Description: "Jin10 (金十数据) market news flash",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("important", "set to 1/true to only include important news"),
	},
	CacheTTL: 5 * time.Minute,
	Handler:  Jin10FlashHandler,
}

// Jin10FlashHandler handles /jin10
func Jin10FlashHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var resp jin10Resp
	url := "https://flash-api.jin10.com/get_flash_list?channel=-8200&vip=1"
	if err := jin10Profile.Fetch(url).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	var flashes []jin10Flash
	if err := json.Unmarshal(resp.Data, &flashes); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("金十数据", "https://www.jin10.com/", "金十数据市场快讯")
	if routeutils.ParseBool(c.QueryParam("important"), false) {
		filtered := make([]jin10Flash, 0, len(flashes))
		for _, f := range flashes {
			if f.Important != 0 {
				filtered = append(filtered, f)
			}
		}
		flashes = filtered
	}
	mapJin10Flashes(feed, flashes, "jin10:index")
	return feed, nil
}

var jin10CategoryRoute = routeutils.RouteSpec{
	Path:        "category/:id",
	Name:        "Jin10 Category News",
	Example:     "jin10/category/36",
	Maintainers: []string{"xihale"},
	Description: "Jin10 (金十数据) classified market news, e.g. 36=期货 12=外汇 2=黄金",
	Categories:  []models.Category{{Name: "finance"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "category id from jin10 classification list"),
	},
	CacheTTL: 5 * time.Minute,
	Handler:  Jin10CategoryHandler,
}

// Jin10CategoryHandler handles /jin10/category/:id
func Jin10CategoryHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	url := fmt.Sprintf("https://4a735ea38f8146198dc205d2e2d1bd28.z3c.jin10.com/flash?channel=-8200&vip=1&classify=%%5B%s%%5D", id)
	var resp jin10Resp
	if err := jin10Z3cProfile.Fetch(url).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	var flashes []jin10Flash
	if err := json.Unmarshal(resp.Data, &flashes); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("金十数据", "https://www.jin10.com/", "金十数据分类快讯")
	mapJin10Flashes(feed, flashes, "jin10:category")
	return feed, nil
}
