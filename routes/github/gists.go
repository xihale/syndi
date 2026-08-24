package routes

import (
	"fmt"
	"html"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitHubGistsRoute = routeutils.RouteSpec{
	Path:        "gists/:user",
	Name:        "GitHub User Gists",
	Example:     "github/gists/torvalds",
	Maintainers: []string{"xihale"},
	Description: "Fetch public gists from a GitHub user",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("user", "GitHub username"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitHubGistsHandler,
}

// GitHubGistsHandler handles /github/gists/:user
func GitHubGistsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	user := c.Param("user")
	ctx := c.Parent()

	url := fmt.Sprintf("https://api.github.com/users/%s/gists?per_page=30", user)

	var gists []GitHubGist
	if err := routeutils.GetJSON(ctx, c.Client(), url, &gists); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s Gists", user),
		fmt.Sprintf("https://gist.github.com/%s", user),
		fmt.Sprintf("Public gists from %s", user),
	)
	routeutils.AppendMappedItems(feed, gists, 30, func(gist GitHubGist) *models.Item {
		if gist.ID == "" {
			return nil
		}
		link := fmt.Sprintf("https://gist.github.com/%s/%s", user, gist.ID)

		title := gist.Description
		filenames := make([]string, 0, len(gist.Files))
		for name := range gist.Files {
			filenames = append(filenames, name)
		}
		if title == "" && len(filenames) > 0 {
			title = filenames[0]
		}
		if title == "" {
			title = fmt.Sprintf("Gist %s", gist.ID)
		}

		description := ""
		if gist.Description != "" {
			description = html.EscapeString(gist.Description)
		} else if len(filenames) > 0 {
			description = html.EscapeString(filenames[0])
		}

		item := routeutils.NewItem(title, link, description, gist.CreatedAt)
		item.GUID = gist.ID
		routeutils.SetCategories(item, filenames...)
		return item
	})

	return feed, nil
}

type GitHubGist struct {
	ID          string                    `json:"id"`
	Description string                    `json:"description"`
	CreatedAt   time.Time                 `json:"created_at"`
	Files       map[string]GitHubGistFile `json:"files"`
}

type GitHubGistFile struct {
	Filename string `json:"filename"`
	RawURL   string `json:"raw_url"`
}
