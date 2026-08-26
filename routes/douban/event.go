package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var doubanEventHotRoute = routeutils.RouteSpec{
	Path:        "event/hot/:locationId",
	Name:        "Hot Douban City Events",
	Example:     "douban/event/hot/118172",
	Maintainers: []string{"xihale"},
	Description: "Hot douban city events for a location (热门同城活动); location id is available as window.__loc_id__ on www.douban.com/location",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("locationId", "Douban city location id, e.g. 118172 (Hangzhou)"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  DoubanEventHotHandler,
}

// DoubanEventHotHandler handles /douban/event/hot/:locationId
func DoubanEventHotHandler(c *ctxpkg.Context) (*models.Feed, error) {
	locationID := doubanSanitizeKey(c.Param("locationId"), "")
	if locationID == "" {
		return nil, fmt.Errorf("douban event/hot: invalid locationId %q", c.Param("locationId"))
	}
	ctx := c.Parent()

	referer := doubanMobileURL + "/app_topic/event_hot"
	apiURL := fmt.Sprintf("%s/subject_collection/event_hot/items?os=ios&for_mobile=1&callback=&start=0&count=20&loc_id=%s", doubanRexxarAPI, locationID)
	var resp doubanCollectionResp
	if err := doubanFetchJSON(ctx, c.Client(), apiURL, referer, &resp); err != nil {
		return nil, err
	}

	title := fmt.Sprintf("豆瓣同城-热门活动-%s", locationID)
	feed := routeutils.NewFeed(title, referer, title)
	for _, entry := range resp.items() {
		if item := buildDoubanEventItem(entry); item != nil {
			routeutils.AddItem(feed, item)
		}
	}
	return feed, nil
}

func buildDoubanEventItem(entry doubanCollectionItem) *models.Item {
	title := routeutils.CollapseWhitespace(entry.Title)
	link := entry.URL
	if title == "" || link == "" {
		return nil
	}

	var sb strings.Builder
	if poster := entry.Poster(); poster != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s"/><br>`, html.EscapeString(poster)))
	}
	parts := make([]string, 0, 3)
	for _, part := range []string{entry.Info, entry.Subtype, entry.PriceRange} {
		if part = routeutils.CollapseWhitespace(part); part != "" {
			parts = append(parts, part)
		}
	}
	sb.WriteString(html.EscapeString(strings.Join(parts, " / ")))

	item := routeutils.NewItem(title, link, sb.String(), time.Time{})
	if item == nil {
		return nil
	}
	item.GUID = "douban-event-" + firstNonEmpty(entry.ID, doubanIDFromLink(link))
	return item
}
