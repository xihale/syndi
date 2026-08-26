package routes

import (
	"fmt"
	"html"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitHubBranchesRoute = routeutils.RouteSpec{
	Path:        "branches/:owner/:repo",
	Name:        "GitHub Repo Branches",
	Example:     "github/branches/DIYgod/RSSHub",
	Maintainers: []string{"xihale"},
	Description: "Fetch branches of a GitHub repository",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitHubBranchesHandler,
}

// GitHubBranchesHandler handles /github/branches/:owner/:repo
func GitHubBranchesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)
	ctx := c.Parent()

	url := fmt.Sprintf("%s/repos/%s/%s/branches?per_page=%d", gitHubAPIBase, owner, repo, limit)

	var branches []GitHubBranch
	if err := gitHubGetJSON(ctx, c.Client(), url, &branches); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s/%s Branches", owner, repo),
		fmt.Sprintf("%s/%s/%s/branches/all", gitHubWebBase, owner, repo),
		fmt.Sprintf("Latest branches from %s/%s", owner, repo),
	)
	routeutils.AppendMappedItems(feed, branches, limit, func(branch GitHubBranch) *models.Item {
		if branch.Name == "" {
			return nil
		}
		link := fmt.Sprintf("%s/%s/%s/commits/%s", gitHubWebBase, owner, repo, branch.Name)
		description := html.EscapeString(branch.Name)
		if sha := shortSHA(branch); sha != "" {
			description += fmt.Sprintf("<br/>HEAD: %s", sha)
		}
		item := routeutils.NewItem(branch.Name, link, description, time.Time{})
		item.GUID = fmt.Sprintf("%s/%s@%s", owner, repo, branch.Name)
		return item
	})

	return feed, nil
}

func shortSHA(branch GitHubBranch) string {
	if len(branch.Commit.SHA) >= 7 {
		return branch.Commit.SHA[:7]
	}
	return branch.Commit.SHA
}

// GitHubBranch is one entry of the /repos/{owner}/{repo}/branches endpoint.
type GitHubBranch struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
}
