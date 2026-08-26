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

var gitHubFileRoute = routeutils.RouteSpec{
	Path:        "file/:owner/:repo/:branch/*filepath",
	Name:        "GitHub File Commits",
	Example:     "github/file/DIYgod/RSSHub/master/README.md",
	Maintainers: []string{"xihale"},
	Description: "Fetch commit history of a single file on a GitHub repository branch",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
		routeutils.RequiredParam("branch", "Branch name"),
		routeutils.RequiredParam("filepath", "Path of the target file (may contain slashes)"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitHubFileHandler,
}

// GitHubFileHandler handles /github/file/:owner/:repo/:branch/*filepath
func GitHubFileHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	branch := strings.TrimSpace(c.Param("branch"))
	filepath := strings.TrimPrefix(c.Param("filepath"), "/")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)

	if filepath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/commits?sha=%s&path=%s&per_page=%d",
		gitHubAPIBase, owner, repo, url.QueryEscape(branch), url.QueryEscape(filepath), limit)
	ctx := c.Parent()

	var commits []GitHubCommit
	if err := gitHubGetJSON(ctx, c.Client(), endpoint, &commits); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("GitHub File - %s/%s/%s/%s", owner, repo, branch, filepath),
		fmt.Sprintf("%s/%s/%s/commits/%s/%s", gitHubWebBase, owner, repo, branch, filepath),
		fmt.Sprintf("Latest commits touching %s on %s/%s (%s)", filepath, owner, repo, branch),
	)
	routeutils.AppendMappedItems(feed, commits, limit, func(commit GitHubCommit) *models.Item {
		if commit.SHA == "" {
			return nil
		}
		title := firstLine(commit.Commit.Message)
		if title == "" {
			title = commit.SHA[:7]
		}
		link := commit.HTMLURL
		if link == "" {
			link = fmt.Sprintf("%s/%s/%s/commit/%s", gitHubWebBase, owner, repo, commit.SHA)
		}

		description := "<pre>" + html.EscapeString(strings.TrimSpace(commit.Commit.Message)) + "</pre>"

		pubDate := commit.Commit.Committer.Date
		if pubDate.IsZero() {
			pubDate = commit.Commit.Author.Date
		}
		item := routeutils.NewItem(title, link, description, pubDate)
		item.GUID = commit.SHA
		if name := commit.Commit.Author.Name; name != "" {
			routeutils.SetItemAuthor(item, name, commit.Commit.Author.Email, "")
		}
		return item
	})

	return feed, nil
}
