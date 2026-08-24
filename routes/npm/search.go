package routes

import (
	"fmt"
	"html"
	"net/url"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var npmSearchRoute = routeutils.RouteSpec{
	Path:        "search/:keyword",
	Name:        "npm Search",
	Example:     "npm/search/rss",
	Maintainers: []string{"xihale"},
	Description: "Search packages on npm",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("keyword", "Search keyword"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  NPMSearchHandler,
}

// NPMSearchHandler handles /npm/search/:keyword
func NPMSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	keyword := c.Param("keyword")
	ctx := c.Parent()

	apiURL := fmt.Sprintf("https://registry.npmjs.org/-/v1/search?text=%s&size=20", url.QueryEscape(keyword))

	var response NPMSearchResponse
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &response); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("npm search: %s", keyword),
		fmt.Sprintf("https://www.npmjs.com/search?q=%s", url.QueryEscape(keyword)),
		fmt.Sprintf("npm packages matching %s", keyword),
	)
	routeutils.AppendMappedItems(feed, response.Objects, 20, func(obj NPMSearchObject) *models.Item {
		pkg := obj.Package
		if pkg.Name == "" {
			return nil
		}
		link := pkg.Links.NPM
		if link == "" {
			link = fmt.Sprintf("https://www.npmjs.com/package/%s", pkg.Name)
		}

		var published time.Time
		if pkg.Date != "" {
			if t, err := time.Parse(time.RFC3339, pkg.Date); err == nil {
				published = t
			}
		}

		item := routeutils.NewItem(
			fmt.Sprintf("%s@%s", pkg.Name, pkg.Version),
			link,
			html.EscapeString(pkg.Description),
			published,
		)
		item.GUID = fmt.Sprintf("npm-search-%s-%s", pkg.Name, pkg.Version)
		if pkg.Publisher.Username != "" {
			routeutils.SetItemAuthor(item, pkg.Publisher.Username, "", fmt.Sprintf("https://www.npmjs.com/~%s", pkg.Publisher.Username))
		}
		routeutils.SetCategories(item, pkg.Keywords...)
		return item
	})

	return feed, nil
}

type NPMSearchResponse struct {
	Objects []NPMSearchObject `json:"objects"`
}

type NPMSearchObject struct {
	Package NPMSearchPackage `json:"package"`
}

type NPMSearchPackage struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Links       struct {
		NPM        string `json:"npm"`
		Homepage   string `json:"homepage"`
		Repository string `json:"repository"`
	} `json:"links"`
	Publisher struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"publisher"`
}
