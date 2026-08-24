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

// odailyHotResp is the payload of /hotRank/list.
type odailyHotResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		EntityID   odailyInt64 `json:"entityId"`
		EntityType int         `json:"entityType"`
		Title      string      `json:"title"`
		Cover      string      `json:"cover"`
	} `json:"data"`
}

// OdailyHotHandler serves both /odaily/hot (weekly default) and
// /odaily/hot/:period — the site's daily/weekly hot post ranking. The two
// RouteSpecs share this handler because gin has no optional path segments.
func OdailyHotHandler(c *ctxpkg.Context) (*models.Feed, error) {
	period := routeutils.ParseEnum(c.Param("period"), "weekly", "daily", "weekly")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 10, 25)
	ctx := c.Parent()

	hotRankType := "WEEKLY"
	feedTitle := "本周热文"
	if period == "daily" {
		hotRankType = "DAILY"
		feedTitle = "今日热文"
	}

	listURL := fmt.Sprintf("%s/hotRank/list?hotRankType=%s&entityType=POST", odailyAPIBase, hotRankType)
	var resp odailyHotResp
	if err := odailyProfile().Fetch(listURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("odaily: hotRank/list returned code %d (%s)", resp.Code, resp.Msg)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("odaily: no hot posts returned")
	}

	ids := make([]string, 0, len(resp.Data))
	for _, it := range resp.Data {
		if it.EntityID.Int64() > 0 {
			ids = append(ids, it.EntityID.String())
		}
	}
	details := fetchOdailyDetails(ctx, c.Client(), ids)

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s - Odaily星球日报", feedTitle),
		odailyRootURL+"/hot",
		fmt.Sprintf("Odaily 星球日报%s榜", feedTitle),
	)
	for _, it := range resp.Data {
		id := it.EntityID.String()
		title := strings.TrimSpace(it.Title)
		if title == "" || it.EntityID.Int64() == 0 {
			continue
		}
		if len(feed.Items) >= limit {
			break
		}

		desc := ""
		var pubDate time.Time
		authorName := ""
		if d := details[id]; d != nil {
			if strings.TrimSpace(d.Content) != "" {
				desc = d.Content
			}
			pubDate = odailyTime(d.PublishTimestamp)
			authorName = strings.TrimSpace(d.Author.Nickname)
		}
		if desc == "" {
			desc = "<p>" + html.EscapeString(title) + "</p>"
		}
		if cover := odailyCleanImage(it.Cover); cover != "" {
			desc += fmt.Sprintf(`<p><img src="%s"/></p>`, html.EscapeString(cover))
		}

		item := routeutils.NewItem(title, fmt.Sprintf("%s/post/%s", odailyRootURL, id), desc, pubDate)
		if item == nil {
			continue
		}
		item.GUID = "odaily-post-" + id
		if authorName != "" {
			routeutils.SetItemAuthor(item, authorName, "", "")
		}
		routeutils.AddItem(feed, item)
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("odaily: hot feed produced no items")
	}
	return feed, nil
}
