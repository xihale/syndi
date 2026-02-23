package routes

import (
	"context"
	"fmt"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var gitHubReposRoute = routeutils.RouteSpec{
	Path:        "/github/repos/:owner/:repo",
	Name:        "GitHub Repository Releases",
	Example:     "github/repos/DIYgod/RSSHub",
	Maintainers: []string{"xihale"},
	Description: "Fetch releases from a GitHub repository",
	Categories:  []models.Category{{Name: "dev"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
	},
	CacheTTL: 1 * time.Hour, // GitHub releases are infrequent
	Handler:  GitHubReposHandler,
}

// GitHubReposHandler handles /github/repos/:owner/:repo
func GitHubReposHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 100)
	includePrerelease := routeutils.ParseBool(c.QueryParam("include_prerelease"), true)

	ctx, cancel := context.WithTimeout(c.Parent(), 12*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", owner, repo, limit)

	var releases []GitHubRelease
	if err := routeutils.GetJSON(ctx, c.Client(), url, &releases); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		owner+"/"+repo+" Releases",
		"https://github.com/"+owner+"/"+repo+"/releases",
		"Latest releases from "+owner+"/"+repo,
	)
	routeutils.AppendMappedItems(feed, releases, limit, func(release GitHubRelease) *models.Item {
		if release.TagName == "" || release.HTMLURL == "" {
			return nil
		}
		if !includePrerelease && release.Prerelease {
			return nil
		}
		return &models.Item{
			Title:       "Release " + release.TagName,
			Link:        release.HTMLURL,
			GUID:        release.HTMLURL,
			Description: release.Body,
			PubDate:     release.PublishedAt,
		}
	})

	return feed, nil
}

type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Prerelease  bool      `json:"prerelease"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
}
