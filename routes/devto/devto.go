package routes

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var devtoArticlesRoute = routeutils.RouteSpec{
	Path:        "articles",
	Name:        "DEV Community Articles",
	Example:     "devto/articles",
	Maintainers: []string{"xihale"},
	Description: "Latest articles from DEV Community",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    15 * time.Minute,
	Handler:     DevtoArticlesHandler,
}

var devtoTagRoute = routeutils.RouteSpec{
	Path:        "tag/:tag",
	Name:        "DEV Community Tag",
	Example:     "devto/tag/python",
	Maintainers: []string{"xihale"},
	Description: "Latest DEV Community articles for a tag",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("tag", "Tag name, e.g. python"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  DevtoTagHandler,
}

// DevtoArticlesHandler handles /devto/articles
func DevtoArticlesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return fetchDevto(c, "https://dev.to/api/articles?per_page=30", "DEV Community")
}

// DevtoTagHandler handles /devto/tag/:tag
func DevtoTagHandler(c *ctxpkg.Context) (*models.Feed, error) {
	tag := c.Param("tag")
	url := fmt.Sprintf("https://dev.to/api/articles?tag=%s&per_page=30", url.QueryEscape(tag))
	return fetchDevto(c, url, fmt.Sprintf("DEV Community (%s)", tag))
}

func fetchDevto(c *ctxpkg.Context, apiURL, title string) (*models.Feed, error) {
	ctx := c.Parent()

	var articles []devtoArticle
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &articles); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(title, "https://dev.to/", "Latest articles from DEV Community")
	routeutils.AppendMappedItems(feed, articles, 0, func(a devtoArticle) *models.Item {
		if a.Title == "" || a.URL == "" {
			return nil
		}
		item := routeutils.NewItem(
			a.Title,
			a.URL,
			buildDevtoDescription(a),
			a.PublishedAt,
		)
		item.GUID = fmt.Sprintf("devto-%d", a.ID)
		if a.User.Name != "" {
			routeutils.SetAuthor(item, a.User.Name, routeutils.WithAuthorURI("https://dev.to/"+a.User.Username))
		}
		routeutils.SetCategories(item, a.TagList...)
		return item
	})

	return feed, nil
}

func buildDevtoDescription(a devtoArticle) string {
	var sb strings.Builder
	desc := strings.TrimSpace(a.DescriptionOfArticle)
	if desc != "" {
		sb.WriteString(html.EscapeString(desc))
	}
	fmt.Fprintf(&sb, "<br/>Reactions: %d | Comments: %d", a.PositiveReactionsCount, a.CommentsCount)
	if len(a.TagList) > 0 {
		sb.WriteString("<br/>Tags: " + html.EscapeString(strings.Join(a.TagList, ", ")))
	}
	return sb.String()
}

type devtoArticle struct {
	ID                     int       `json:"id"`
	Title                  string    `json:"title"`
	DescriptionOfArticle   string    `json:"description"`
	URL                    string    `json:"url"`
	PublishedAt            time.Time `json:"published_at"`
	PositiveReactionsCount int       `json:"positive_reactions_count"`
	CommentsCount          int       `json:"comments_count"`
	TagList                []string  `json:"tag_list"`
	User                   struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"user"`
}
