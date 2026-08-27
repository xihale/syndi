package routes

import (
	"fmt"
	"net/url"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitHubIssueRoute = routeutils.RouteSpec{
	Path:        "issue/:owner/:repo",
	Name:        "GitHub Repo Issues",
	Example:     "github/issue/xi-editor/xi-editor",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest issues of a GitHub repository, filtered by state (default open)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
		routeutils.OptionalParam("state", "Issue state: open (default), closed or all"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  gitHubIssueHandler,
}

var gitHubIssueStateRoute = routeutils.RouteSpec{
	Path:        "issue/:owner/:repo/:state",
	Name:        "GitHub Repo Issues by State",
	Example:     "github/issue/DIYgod/RSSHub/closed",
	Maintainers: []string{"xihale"},
	Description: "Fetch issues of a GitHub repository with an explicit state (open/closed/all)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
		routeutils.RequiredParam("state", "Issue state: open, closed or all"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  gitHubIssueHandler,
}

var gitHubIssueLabelsRoute = routeutils.RouteSpec{
	Path:        "issue/:owner/:repo/:state/:labels",
	Name:        "GitHub Repo Issues by State and Labels",
	Example:     "github/issue/golang/go/all/Performance",
	Maintainers: []string{"xihale"},
	Description: "Fetch issues of a GitHub repository by state and a comma-separated label list",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
		routeutils.RequiredParam("state", "Issue state: open, closed or all"),
		routeutils.RequiredParam("labels", "Comma-separated label names to filter by"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  gitHubIssueHandler,
}

// gitHubIssueHandler backs all /github/issue/* routes.
func gitHubIssueHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return gitHubIssuesFeed(c, gitHubIssuesFeedOptions{
		Owner:      c.Param("owner"),
		Repo:       c.Param("repo"),
		State:      routeutils.ParseEnum(c.Param("state"), "open", "open", "closed", "all"),
		Labels:     c.Param("labels"),
		Noun:       "Issues",
		ItemPrefix: "gh-issue-",
		WantPulls:  false,
	})
}

// gitHubIssuesFeedOptions configures the shared GitHub issue/pull list feed.
type gitHubIssuesFeedOptions struct {
	Owner string
	Repo  string
	State string // open | closed | all
	// Labels is a comma-separated label filter; empty means no filtering.
	Labels string
	// Noun is the feed heading noun ("Issues" / "Pull Requests").
	Noun string
	// ItemPrefix is used for item GUIDs ("gh-issue-" / "gh-pull-").
	ItemPrefix string
	// WantPulls selects pull requests from the issues endpoint when true;
	// otherwise plain issues are selected.
	WantPulls bool
}

// gitHubIssuesFeed renders one of GitHub's issue-list feeds. Pull requests
// are also issues, so both kinds are served through the same endpoint
// (mirroring the upstream implementation).
func gitHubIssuesFeed(c *ctxpkg.Context, opts gitHubIssuesFeedOptions) (*models.Feed, error) {
	ctx := c.Parent()
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)

	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues?state=%s&sort=created&direction=desc&per_page=%d",
		gitHubAPIBase, opts.Owner, opts.Repo, opts.State, limit)
	if opts.Labels != "" {
		endpoint += "&labels=" + url.QueryEscape(opts.Labels)
	}

	var issues []GitHubIssue
	if err := gitHubGetJSON(ctx, c.Client(), endpoint, &issues); err != nil {
		return nil, err
	}

	stateTitle := upperFirst(opts.State)
	labelsTitle := ""
	if opts.Labels != "" {
		labelsTitle = " - " + opts.Labels
	}
	feed := routeutils.NewFeed(
		fmt.Sprintf("%s/%s %s %s%s", opts.Owner, opts.Repo, stateTitle, opts.Noun, labelsTitle),
		fmt.Sprintf("%s/%s/%s/%s", gitHubWebBase, opts.Owner, opts.Repo, listLinkPath(opts.Noun)),
		fmt.Sprintf("Latest %s %s from %s/%s", opts.State, opts.Noun, opts.Owner, opts.Repo),
	)
	routeutils.AppendMappedItems(feed, issues, limit, func(issue GitHubIssue) *models.Item {
		if issue.Number == 0 || (issue.PullRequest != nil) != opts.WantPulls {
			return nil
		}
		link := issue.HTMLURL
		if link == "" {
			link = fmt.Sprintf("%s/%s/%s/%s/%d", gitHubWebBase, opts.Owner, opts.Repo, listLinkPath(opts.Noun), issue.Number)
		}

		item := routeutils.NewItem(issue.Title, link, excerptBody(issue.Body, 600), issue.CreatedAt)
		item.GUID = fmt.Sprintf("%s%d", opts.ItemPrefix, issue.Number)
		if issue.User != nil && issue.User.Login != "" {
			routeutils.SetItemAuthor(item, issue.User.Login, "", fmt.Sprintf("%s/%s", gitHubWebBase, issue.User.Login))
		}
		for _, label := range issue.Labels {
			routeutils.SetCategories(item, label.Name)
		}
		return item
	})
	return feed, nil
}

func listLinkPath(noun string) string {
	if noun == "Pull Requests" {
		return "pulls"
	}
	return "issues"
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 'a' - 'A'
	}
	return string(b)
}
