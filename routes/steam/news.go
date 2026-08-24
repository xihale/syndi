package routes

import (
	"context"
	"fmt"
	"html"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var steamNewsRoute = routeutils.RouteSpec{
	Path:        "news/:appid",
	Name:        "Steam Game News",
	Example:     "steam/news/570",
	Maintainers: []string{"xihale"},
	Description: "Latest news, updates and events for a Steam game by appid",
	Categories:  []models.Category{{Name: "game"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("appid", "Steam appid (e.g. 570 for Dota 2)"),
		routeutils.OptionalParam("limit", "Maximum number of news items (default 20, max 20)"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  SteamNewsHandler,
}

// steamNewsResponse mirrors https://store.steampowered.com/api/appnews
type steamNewsResponse struct {
	AppNews struct {
		AppID     int             `json:"appid"`
		NewsItems []steamNewsItem `json:"newsitems"`
	} `json:"appnews"`
}

type steamNewsItem struct {
	GID       string `json:"gid"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Author    string `json:"author"`
	Contents  string `json:"contents"`
	FeedLabel string `json:"feedlabel"`
	Date      int64  `json:"date"` // unix epoch seconds
}

// SteamNewsHandler handles /steam/news/:appid
func SteamNewsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	appid := c.Param("appid")
	if appid == "" {
		return nil, fmt.Errorf("appid is required")
	}
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 20)

	ctx, cancel := context.WithTimeout(c.Parent(), 30*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://store.steampowered.com/api/appnews/?appid=%s&count=%d", appid, limit)

	var resp steamNewsResponse
	if err := routeutils.GetJSON(ctx, c.Client(), url, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Steam News - %s", appid),
		fmt.Sprintf("https://store.steampowered.com/news/?appids=%s", appid),
		fmt.Sprintf("Latest news for Steam app %s", appid),
	)
	routeutils.AppendMappedItems(feed, resp.AppNews.NewsItems, limit, func(n steamNewsItem) *models.Item {
		if n.Title == "" || n.URL == "" {
			return nil
		}
		description := n.Contents
		if description == "" {
			description = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(n.URL), "Read the full announcement")
		}
		item := routeutils.NewItem(n.Title, n.URL, description, time.Unix(n.Date, 0))
		if item == nil {
			return nil
		}
		item.GUID = n.GID
		if n.Author != "" {
			routeutils.SetItemAuthor(item, n.Author, "", "")
		}
		if n.FeedLabel != "" {
			routeutils.SetCategories(item, n.FeedLabel)
		}
		return item
	})

	return feed, nil
}
