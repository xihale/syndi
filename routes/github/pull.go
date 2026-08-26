package routes

import (
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

var gitHubPullStateRoute = routeutils.RouteSpec{
	Path:        "pull/:owner/:repo/:state",
	Name:        "GitHub Repository Pull Requests by State",
	Example:     "github/pull/gin-gonic/gin/closed",
	Maintainers: []string{"xihale"},
	Description: "Fetch pull requests of a GitHub repository with an explicit state (open/closed/all)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
		routeutils.RequiredParam("state", "Pull request state: open, closed or all"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitHubPullHandler,
}

var gitHubPullLabelsRoute = routeutils.RouteSpec{
	Path:        "pull/:owner/:repo/:state/:labels",
	Name:        "GitHub Repository Pull Requests by State and Labels",
	Example:     "github/pull/gin-gonic/gin/all/performance",
	Maintainers: []string{"xihale"},
	Description: "Fetch pull requests of a GitHub repository by state and a comma-separated label list",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
		routeutils.OptionalParam("state", "Pull request state: open (default), closed or all"),
		routeutils.OptionalParam("labels", "Comma-separated label names to filter by"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitHubPullHandler,
}

// GitHubPullHandler handles /github/pull/:owner/:repo[/:state[/:labels]].
// Every pull request is also an issue, so the list is served through the
// issues endpoint filtered down to PRs (mirroring the upstream behavior).
func GitHubPullHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return gitHubIssuesFeed(c, gitHubIssuesFeedOptions{
		Owner:      c.Param("owner"),
		Repo:       c.Param("repo"),
		State:      routeutils.ParseEnum(c.Param("state"), "open", "open", "closed", "all"),
		Labels:     c.Param("labels"),
		Noun:       "Pull Requests",
		ItemPrefix: "gh-pull-",
		WantPulls:  true,
	})
}
