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

var gitHubIssuesRoute = routeutils.RouteSpec{
	Path:        "issues/:owner/:repo",
	Name:        "GitHub Repository Issues",
	Example:     "github/issues/golang/go",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest open issues from a GitHub repository",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  GitHubIssuesHandler,
}

// GitHubIssuesHandler handles /github/issues/:owner/:repo
func GitHubIssuesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	ctx := c.Parent()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=open&per_page=30&sort=created", owner, repo)

	var issues []GitHubIssue
	if err := routeutils.GetJSON(ctx, c.Client(), url, &issues); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s/%s Issues", owner, repo),
		fmt.Sprintf("https://github.com/%s/%s/issues", owner, repo),
		fmt.Sprintf("Latest open issues from %s/%s", owner, repo),
	)
	routeutils.AppendMappedItems(feed, issues, 30, func(issue GitHubIssue) *models.Item {
		// The issues endpoint also returns pull requests; exclude them.
		if issue.PullRequest != nil || issue.Number == 0 {
			return nil
		}
		link := issue.HTMLURL
		if link == "" {
			link = fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, issue.Number)
		}

		item := routeutils.NewItem(issue.Title, link, excerptBody(issue.Body, 300), issue.CreatedAt)
		item.GUID = link
		if issue.User != nil && issue.User.Login != "" {
			routeutils.SetItemAuthor(item, issue.User.Login, "", fmt.Sprintf("https://github.com/%s", issue.User.Login))
		}
		for _, label := range issue.Labels {
			routeutils.SetCategories(item, label.Name)
		}
		return item
	})

	return feed, nil
}

// excerptBody returns the first maxChars of body as escaped HTML.
func excerptBody(body string, maxChars int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if len(body) > maxChars {
		body = strings.TrimSpace(body[:maxChars]) + "..."
	}
	return html.EscapeString(body)
}

type GitHubIssue struct {
	Number    int         `json:"number"`
	Title     string      `json:"title"`
	HTMLURL   string      `json:"html_url"`
	CreatedAt time.Time   `json:"created_at"`
	User      *GitHubUser `json:"user"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Body        string    `json:"body"`
	PullRequest *struct{} `json:"pull_request"`
}
