package routes

import (
	"fmt"
	"html"
	"strings"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// PixivUserHandler serves /pixiv/user/:id — a user's latest artworks.
//
// The profile/all endpoint returns every artwork id but no metadata, so we
// take the newest ids in document order and fetch each /ajax/illust/{id}
// for title, caption, tags, stats and upload date (bounded concurrency).
func PixivUserHandler(c *ctxpkg.Context) (*models.Feed, error) {
	if err := requirePixivCookies(); err != nil {
		return nil, err
	}
	id := c.Param("id")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 10, 30)
	ctx := c.Parent()

	profileURL := fmt.Sprintf("%s/ajax/user/%s/profile/all?lang=en", pixivBaseURL, id)
	req := pixivProfile(fmt.Sprintf("%s/users/%s", pixivBaseURL, id)).Fetch(profileURL)
	raw, err := req.GetBytes(ctx, c.Client())
	if err != nil {
		return nil, err
	}
	allIDs, err := orderedProfileIllustIDs(raw)
	if err != nil {
		return nil, err
	}
	if len(allIDs) == 0 {
		return nil, fmt.Errorf("pixiv: user %s has no visible illusts (cookies may be expired)", id)
	}
	if len(allIDs) > limit {
		allIDs = allIDs[:limit]
	}

	details := fetchPixivIllustDetails(ctx, c.Client(), allIDs)

	feedTitle := fmt.Sprintf("User %s 的 pixiv 动态", id)
	feedDesc := fmt.Sprintf("User %s 的 pixiv 最新插画", id)
	userPage := fmt.Sprintf("%s/users/%s", pixivBaseURL, id)
	if d := details[allIDs[0]]; d != nil && d.UserName != "" {
		feedTitle = fmt.Sprintf("%s 的 pixiv 动态", d.UserName)
		feedDesc = fmt.Sprintf("%s 的 pixiv 最新动态", d.UserName)
	}
	feed := routeutils.NewFeed(feedTitle, userPage, feedDesc)

	for _, illustID := range allIDs {
		d := details[illustID]
		if d == nil || strings.TrimSpace(d.Title) == "" {
			continue
		}
		link := fmt.Sprintf("%s/artworks/%s", pixivBaseURL, illustID)

		var b strings.Builder
		b.WriteString(d.Comment)
		fmt.Fprintf(&b, "<br/><p>画师：%s - 阅览数：%d - 收藏数：%d</p>",
			html.EscapeString(d.UserName), d.ViewCount, d.BookmarkCount)
		fmt.Fprintf(&b, `<p><img src="%s"/></p>`, html.EscapeString(pixivEmbedImageURL(illustID)))

		item := routeutils.NewItem(d.Title, link, b.String(), d.CreateDate.Time)
		if item == nil {
			continue
		}
		item.GUID = "pixiv-illust-" + illustID
		routeutils.SetItemAuthor(item, d.UserName, "", fmt.Sprintf("%s/users/%s", pixivBaseURL, d.UserID))
		tags := make([]string, 0, len(d.Tags.Tags))
		for _, t := range d.Tags.Tags {
			if t.Tag != "" {
				tags = append(tags, t.Tag)
			}
		}
		routeutils.SetCategories(item, tags...)
		routeutils.AddItem(feed, item)
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("pixiv: no illust details fetched for user %s", id)
	}
	return feed, nil
}
