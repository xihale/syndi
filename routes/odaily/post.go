package routes

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// OdailyPostHandler serves /odaily — the latest featured posts, enriched with
// full article HTML from /post/detail/{id}.
func OdailyPostHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 15, 25)
	ctx := c.Parent()

	listURL := fmt.Sprintf("%s/post/page?page=1&size=%d", odailyAPIBase, limit)
	var page odailyPageResp
	if err := odailyProfile().Fetch(listURL).GetJSON(ctx, c.Client(), &page); err != nil {
		return nil, err
	}
	if !page.Success && page.Code != 200 {
		return nil, fmt.Errorf("odaily: post/page returned code %d (%s)", page.Code, page.Msg)
	}
	if len(page.Data.List) == 0 {
		return nil, fmt.Errorf("odaily: no posts returned")
	}

	ids := make([]string, 0, len(page.Data.List))
	for _, it := range page.Data.List {
		ids = append(ids, it.ID.String())
	}
	details := fetchOdailyDetails(ctx, c.Client(), ids)

	feed := routeutils.NewFeed(
		"文章 - Odaily星球日报",
		odailyRootURL,
		"Odaily 星球日报最新文章",
	)
	for _, it := range page.Data.List {
		title := strings.TrimSpace(it.Title)
		if title == "" || it.ID.Int64() == 0 {
			continue
		}

		desc := ""
		pubDate := odailyTime(it.PublishTimestamp)
		authorName := ""
		if d := details[it.ID.String()]; d != nil {
			if strings.TrimSpace(d.Content) != "" {
				desc = d.Content
			}
			if d.PublishTimestamp.Int64() > it.PublishTimestamp.Int64() {
				pubDate = odailyTime(d.PublishTimestamp)
			}
			authorName = strings.TrimSpace(d.Author.Nickname)
		}
		if desc == "" {
			summary := strings.TrimSpace(it.Summary)
			if summary == "" {
				continue
			}
			desc = "<p>" + html.EscapeString(summary) + "</p>"
		}
		if cover := odailyCleanImage(it.Cover); cover != "" {
			desc += fmt.Sprintf(`<p><img src="%s"/></p>`, html.EscapeString(cover))
		}

		item := routeutils.NewItem(title, fmt.Sprintf("%s/post/%s", odailyRootURL, it.ID), desc, pubDate)
		if item == nil {
			continue
		}
		item.GUID = "odaily-post-" + strconv.FormatInt(it.ID.Int64(), 10)
		if authorName != "" {
			routeutils.SetItemAuthor(item, authorName, "", "")
		}
		tagNames := make([]string, 0, len(it.Tags))
		for _, t := range it.Tags {
			if t.Name != "" {
				tagNames = append(tagNames, t.Name)
			}
		}
		routeutils.SetCategories(item, tagNames...)
		routeutils.AddItem(feed, item)
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("odaily: feed produced no items")
	}
	return feed, nil
}
