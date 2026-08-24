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

var gitHubCommitsRoute = routeutils.RouteSpec{
	Path:        "commits/:owner/:repo",
	Name:        "GitHub Repository Commits",
	Example:     "github/commits/gin-gonic/gin",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest commits from a GitHub repository",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  GitHubCommitsHandler,
}

// GitHubCommitsHandler handles /github/commits/:owner/:repo
func GitHubCommitsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	ctx := c.Parent()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?per_page=30", owner, repo)

	var commits []GitHubCommit
	if err := routeutils.GetJSON(ctx, c.Client(), url, &commits); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s/%s Commits", owner, repo),
		fmt.Sprintf("https://github.com/%s/%s/commits", owner, repo),
		fmt.Sprintf("Latest commits from %s/%s", owner, repo),
	)
	routeutils.AppendMappedItems(feed, commits, 30, func(commit GitHubCommit) *models.Item {
		if commit.SHA == "" {
			return nil
		}
		title := firstLine(commit.Commit.Message)
		if title == "" {
			title = commit.SHA[:7]
		}
		link := commit.HTMLURL
		if link == "" {
			link = fmt.Sprintf("https://github.com/%s/%s/commit/%s", owner, repo, commit.SHA)
		}

		description := html.EscapeString(strings.TrimSpace(commit.Commit.Message))
		description = strings.ReplaceAll(description, "\n", "<br/>")

		item := routeutils.NewItem(title, link, description, commit.Commit.Author.Date)
		item.GUID = commit.SHA
		if commit.Author != nil && commit.Author.Login != "" {
			routeutils.SetItemAuthor(item, commit.Author.Login, "", fmt.Sprintf("https://github.com/%s", commit.Author.Login))
			routeutils.SetCategories(item, commit.Author.Login)
		}
		return item
	})

	return feed, nil
}

type GitHubCommit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Message string `json:"message"`
		Author  struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
	Author *GitHubUser `json:"author"`
}

type GitHubUser struct {
	Login     string `json:"login"`
	HTMLURL   string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
