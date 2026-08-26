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

var doubanMovieComingRoute = routeutils.RouteSpec{
	Path:        "movie/coming",
	Name:        "Coming Soon Movies",
	Example:     "douban/movie/coming",
	Maintainers: []string{"xihale"},
	Description: "Movies coming soon to Chinese cinemas (电影即将上映)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Maximum number of items, default 20"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  DoubanMovieComingHandler,
}

// DoubanMovieComingHandler handles /douban/movie/coming
func DoubanMovieComingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()

	apiURL := fmt.Sprintf("%s/movie/coming_soon?start=0&count=%d", doubanRexxarAPI, limit)
	var resp doubanCollectionResp
	if err := doubanFetchJSON(ctx, c.Client(), apiURL, doubanMobileURL+"/movie/", &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"豆瓣电影-即将上映",
		doubanBaseURL+"/coming",
		"即将上映的电影",
	)
	routeutils.AppendMappedItems(feed, resp.items(), limit, buildDoubanComingItem)
	return feed, nil
}

// buildDoubanComingItem renders a coming_soon subject as a feed item,
// mirroring the upstream description layout.
func buildDoubanComingItem(subject doubanCollectionItem) *models.Item {
	title := routeutils.CollapseWhitespace(subject.Title)
	if title == "" {
		return nil
	}
	link := doubanLink(&subject, doubanBaseURL+"/subject/")
	if link == "" {
		return nil
	}

	var sb strings.Builder
	if poster := subject.Poster(); poster != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s" alt="%s" referrerpolicy="no-referrer"/>`, html.EscapeString(poster), html.EscapeString(title)))
	}
	sb.WriteString("<h2>电影信息</h2><ul>")
	writeDoubanNameList := func(label string, names []doubanName) {
		parts := make([]string, 0, len(names))
		for _, n := range names {
			if n.Name != "" {
				parts = append(parts, n.Name)
			}
		}
		if len(parts) > 0 {
			fmt.Fprintf(&sb, "<li>%s：%s</li>", label, html.EscapeString(strings.Join(parts, ", ")))
		}
	}
	writeDoubanNameList("导演", subject.Directors)
	writeDoubanNameList("演员", subject.Actors)
	if len(subject.Genres) > 0 {
		fmt.Fprintf(&sb, "<li>类型：%s</li>", html.EscapeString(strings.Join(subject.Genres, " / ")))
	}
	if len(subject.Pubdate) > 0 {
		fmt.Fprintf(&sb, "<li>上映日期：%s</li>", html.EscapeString(strings.Join(subject.Pubdate, " / ")))
	}
	if subject.WishCount > 0 {
		fmt.Fprintf(&sb, "<li>想看：%.0f</li>", subject.WishCount)
	}
	sb.WriteString("</ul>")
	if intro := strings.TrimSpace(subject.Intro); intro != "" {
		intro = html.EscapeString(intro)
		intro = strings.ReplaceAll(intro, "\n", "<br>")
		sb.WriteString("<h2>剧情简介</h2><p>" + intro + "</p>")
	}

	item := routeutils.NewItem(title, link, sb.String(), doubanParsePubdate(subject.Pubdate))
	if item == nil {
		return nil
	}
	item.GUID = "douban-coming-" + firstNonEmpty(subject.ID, doubanIDFromLink(link))
	routeutils.SetCategories(item, subject.Genres...)
	return item
}
