package routes

import (
	"fmt"
	"html"
	"strings"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// OdailyNewsflashHandler serves /odaily/newsflash — rolling crypto newsflashes.
func OdailyNewsflashHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)
	ctx := c.Parent()

	listURL := fmt.Sprintf("%s/newsflash/page?page=1&size=%d", odailyAPIBase, limit)
	var page odailyPageResp
	if err := odailyProfile().Fetch(listURL).GetJSON(ctx, c.Client(), &page); err != nil {
		return nil, err
	}
	if !page.Success && page.Code != 200 {
		return nil, fmt.Errorf("odaily: newsflash/page returned code %d (%s)", page.Code, page.Msg)
	}

	feed := routeutils.NewFeed(
		"快讯 - Odaily星球日报",
		odailyRootURL+"/newsflash",
		"Odaily 星球日报快讯",
	)
	for _, it := range page.Data.List {
		title := strings.TrimSpace(it.Title)
		if title == "" || it.ID.Int64() == 0 {
			continue
		}
		link := strings.TrimSpace(it.NewsURL)
		if !strings.HasPrefix(link, "http") {
			link = fmt.Sprintf("%s/newsflash/%s", odailyRootURL, it.ID)
		}

		desc := strings.TrimSpace(it.Description)
		if desc != "" {
			desc = "<p>" + desc + "</p>"
		} else {
			desc = "<p>" + html.EscapeString(title) + "</p>"
		}
		if it.NewsURL != "" && it.NewsURL == link {
			desc += fmt.Sprintf(`<p><a href="%s">来源链接</a></p>`, html.EscapeString(link))
		}

		item := routeutils.NewItem(title, link, desc, odailyTime(it.PublishTimestamp))
		if item == nil {
			continue
		}
		item.GUID = "odaily-newsflash-" + it.ID.String()
		routeutils.AddItem(feed, item)
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("odaily: no newsflashes returned")
	}
	return feed, nil
}
