package routes

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var gitLabReleasesRoute = routeutils.RouteSpec{
	Path:        "releases/*project",
	Name:        "GitLab Project Releases",
	Example:     "gitlab/releases/gitlab-org/gitlab-runner",
	Maintainers: []string{"xihale"},
	Description: "Fetch releases from a GitLab project",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("project", "Full project path including namespace, e.g. gitlab-org/gitlab-runner"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitLabReleasesHandler,
}

// GitLabReleasesHandler handles /gitlab/releases/*project
func GitLabReleasesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	projectPath := strings.Trim(c.Param("project"), "/")
	if projectPath == "" {
		return nil, fmt.Errorf("project parameter is required, e.g. gitlab-org/gitlab-runner")
	}
	ctx := c.Parent()

	apiURL := fmt.Sprintf(
		"https://gitlab.com/api/v4/projects/%s/releases?per_page=20",
		url.PathEscape(projectPath),
	)

	var releases []GitLabRelease
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &releases); err != nil {
		return nil, err
	}

	projectURL := fmt.Sprintf("https://gitlab.com/%s", projectPath)
	feed := routeutils.NewFeed(
		fmt.Sprintf("%s Releases - GitLab", projectPath),
		fmt.Sprintf("%s/-/releases", projectURL),
		fmt.Sprintf("Latest releases from GitLab project %s", projectPath),
	)
	routeutils.AppendMappedItems(feed, releases, 20, func(release GitLabRelease) *models.Item {
		if release.TagName == "" {
			return nil
		}
		link := release.Links.WebURL
		if link == "" {
			link = fmt.Sprintf("%s/-/releases/%s", projectURL, url.PathEscape(release.TagName))
		}

		description := release.DescriptionHTML
		if strings.TrimSpace(description) == "" {
			description = fmt.Sprintf("Release %s", html.EscapeString(release.TagName))
		}

		item := routeutils.NewItem(release.TagName, link, description, release.ReleasedAt)
		item.GUID = link
		routeutils.SetCategories(item, release.TagName)
		return item
	})

	return feed, nil
}

type GitLabRelease struct {
	TagName         string    `json:"tag_name"`
	ReleasedAt      time.Time `json:"released_at"`
	DescriptionHTML string    `json:"description_html"`
	Links           struct {
		Self   string `json:"self"`
		WebURL string `json:"web_url"`
	} `json:"_links"`
}
