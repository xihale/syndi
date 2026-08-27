package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var doubanChannelRoute = routeutils.RouteSpec{
	Path:        "channel/:id",
	Name:        "Douban Channel",
	Example:     "douban/channel/30168934",
	Maintainers: []string{"xihale"},
	Description: "Posts of a douban channel topic (频道专题)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Channel id, e.g. 30168934"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  DoubanChannelHandler,
}

var doubanChannelNavRoute = routeutils.RouteSpec{
	Path:        "channel/:id/:nav",
	Name:        "Douban Channel by Nav",
	Example:     "douban/channel/30168934/hot",
	Maintainers: []string{"xihale"},
	Description: "Posts of a douban channel topic (频道专题)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Channel id, e.g. 30168934"),
		routeutils.RequiredParam("nav", "Nav tab: default (默认), hot (热门), new (最新)"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  DoubanChannelHandler,
}

var doubanChannelNavNames = map[string]string{
	"default": "默认",
	"hot":     "热门",
	"new":     "最新",
}

type doubanChannelInfoResp struct {
	Title string `json:"title"`
}

type doubanChannelFeedResp struct {
	Title string              `json:"title"`
	Items []doubanChannelItem `json:"items"`
}

type doubanChannelItem struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Abstract   string `json:"abstract"`
	CreateTime string `json:"create_time"`
	URL        string `json:"url"`
	CoverURL   string `json:"cover_url"`

	Author struct {
		Name string `json:"name"`
	} `json:"author"`

	// ExternalPayload aggregates collection cards; upstream skips entries
	// that carry such nested item lists.
	ExternalPayload *struct {
		Items json.RawMessage `json:"items"`
	} `json:"external_payload"`
}

// DoubanChannelHandler handles /douban/channel/:id/:nav?
func DoubanChannelHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := doubanSanitizeKey(c.Param("id"), "")
	if id == "" {
		return nil, fmt.Errorf("douban channel: invalid id %q", c.Param("id"))
	}
	nav := routeutils.ParseEnum(doubanSanitizeKey(c.Param("nav"), ""), "default", "default", "hot", "new")
	ctx := c.Parent()
	cl := c.Client()

	link := fmt.Sprintf("%s/channel/%s", doubanWWWBaseURL, id)
	referer := link
	channelName := id

	// Channel metadata is decoration only; fall back to the raw id.
	var info doubanChannelInfoResp
	if err := doubanFetchJSON(ctx, cl, fmt.Sprintf("%s/elessar/channel/%s", doubanRexxarAPI, id), referer, &info); err == nil && info.Title != "" {
		channelName = routeutils.CollapseWhitespace(info.Title)
	}

	apiURL := fmt.Sprintf("%s/lembas/channel/%s/feed?ck=null&for_mobile=1&start=0&count=20&nav=%s", doubanRexxarAPI, id, nav)
	var feedResp doubanChannelFeedResp
	if err := doubanFetchJSON(ctx, cl, apiURL, referer, &feedResp); err != nil {
		return nil, err
	}

	navName := doubanChannelNavNames[nav]
	feed := routeutils.NewFeed(
		fmt.Sprintf("豆瓣%s频道-%s动态", channelName, navName),
		link,
		fmt.Sprintf("豆瓣%s频道专题下的%s动态", channelName, navName),
	)
	for _, raw := range feedResp.Items {
		if item := buildDoubanChannelItem(raw); item != nil {
			routeutils.AddItem(feed, item)
		}
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("douban channel: no items for channel %s", id)
	}
	return feed, nil
}

func buildDoubanChannelItem(raw doubanChannelItem) *models.Item {
	if raw.ExternalPayload != nil && len(raw.ExternalPayload.Items) > 0 {
		return nil
	}
	title := routeutils.CollapseWhitespace(firstNonEmpty(raw.Title, routeutils.Truncate(routeutils.CollapseWhitespace(raw.Abstract), 40, "…")))
	link := raw.URL
	if title == "" || link == "" {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("作者：%s | %s<br><br>", html.EscapeString(raw.Author.Name), html.EscapeString(raw.CreateTime)))
	if poster := raw.CoverURL; poster != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s"/><br>`, html.EscapeString(poster)))
	}
	sb.WriteString(html.EscapeString(routeutils.CollapseWhitespace(raw.Abstract)))

	item := routeutils.NewItem(title, link, sb.String(), doubanParseDate(raw.CreateTime))
	if item == nil {
		return nil
	}
	if raw.Author.Name != "" {
		routeutils.SetAuthor(item, raw.Author.Name)
	}
	item.GUID = "douban-channel-" + firstNonEmpty(raw.ID, doubanIDFromLink(link))
	return item
}
