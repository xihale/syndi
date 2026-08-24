package routes

import (
	"fmt"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitHubPullRoute = routeutils.RouteSpec{
	Path:        "pull/:owner/:repo",
	Name:        "GitHub Repository Pull Requests",
	Example:     "github/pull/gin-gonic/gin",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest open pull requests from a GitHub repository",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  GitHubPullHandler,
}

// GitHubPullHandler handles /github/pull/:owner/:repo
func GitHubPullHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	ctx := c.Parent()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=open&per_page=30", owner, repo)

	var pulls []GitHubPull
	if err := routeutils.GetJSON(ctx, c.Client(), url, &pulls); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s/%s Pull Requests", owner, repo),
		fmt.Sprintf("https://github.com/%s/%s/pulls", owner, repo),
		fmt.Sprintf("Latest open pull requests from %s/%s", owner, repo),
	)
	routeutils.AppendMappedItems(feed, pulls, 30, func(pull GitHubPull) *models.Item {
		if pull.Number == 0 {
			return nil
		}
		link := pull.HTMLURL
		if link == "" {
			link = fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, pull.Number)
		}

		item := routeutils.NewItem(pull.Title, link, excerptBody(pull.Body, 300), pull.CreatedAt)
		item.GUID = link
		if pull.User != nil && pull.User.Login != "" {
			routeutils.SetItemAuthor(item, pull.User.Login, "", fmt.Sprintf("https://github.com/%s", pull.User.Login))
		}
		for _, label := range pull.Labels {
			routeutils.SetCategories(item, label.Name)
		}
		return item
	})

	return feed, nil
}

type GitHubPull struct {
	Number    int         `json:"number"`
	Title     string      `json:"title"`
	HTMLURL   string      `json:"html_url"`
	CreatedAt time.Time   `json:"created_at"`
	User      *GitHubUser `json:"user"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Body string `json:"body"`
}
