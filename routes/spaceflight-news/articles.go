package routes

import (
	"fmt"
	"html"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var spaceflightNewsRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Spaceflight News",
	Example:     "spaceflight-news",
	Maintainers: []string{"xihale"},
	Description: "Latest spaceflight news articles from the Spaceflight News API",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     SpaceflightNewsHandler,
}

// SpaceflightNewsHandler handles /spaceflight-news
func SpaceflightNewsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var resp sfnResponse
	if err := routeutils.GetJSON(c.Parent(), c.Client(), "https://api.spaceflightnewsapi.net/v4/articles/?limit=20", &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"Spaceflight News",
		"https://spaceflightnewsapi.net/",
		"Latest spaceflight news from around the web, via the Spaceflight News API",
	)

	routeutils.AppendMappedItems(feed, resp.Results, 20, func(a sfnArticle) *models.Item {
		if a.Title == "" || a.URL == "" {
			return nil
		}

		desc := ""
		if a.ImageURL != "" {
			desc += fmt.Sprintf(`<img src="%s"/><br/>`, html.EscapeString(a.ImageURL))
		}
		if a.Summary != "" {
			desc += "<p>" + html.EscapeString(a.Summary) + "</p>"
		}
		desc += "Source: " + html.EscapeString(a.NewsSite)

		item := routeutils.NewItem(a.Title, a.URL, desc, a.PublishedAt)
		if a.ID != 0 {
			item.GUID = fmt.Sprintf("spaceflight-news-%d", a.ID)
		}
		for _, author := range a.Authors {
			if author.Name != "" && author.Name != "null" {
				routeutils.SetItemAuthor(item, author.Name, "", "")
				break
			}
		}
		return item
	})
	return feed, nil
}

type sfnResponse struct {
	Count   int          `json:"count"`
	Results []sfnArticle `json:"results"`
}

type sfnArticle struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	ImageURL    string    `json:"image_url"`
	NewsSite    string    `json:"news_site"`
	Summary     string    `json:"summary"`
	PublishedAt time.Time `json:"published_at"`
	Authors     []struct {
		Name string `json:"name"`
	} `json:"authors"`
}
