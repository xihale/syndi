package routes

import (
	"fmt"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitHubStarsRoute = routeutils.RouteSpec{
	Path:        "stars/:owner/:repo",
	Name:        "GitHub Repo Releases",
	Example:     "github/stars/gin-gonic/gin",
	Maintainers: []string{"xihale"},
	Description: "Track new releases of a GitHub repository (falls back to tags when the repo has no releases)",
	Categories:  []models.Category{{Name: "dev"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
	},
	CacheTTL: 1 * time.Hour, // Anonymous GitHub API quota is 60 requests/hour
	Handler:  GitHubStarsHandler,
}

// GitHubStarsHandler handles /github/stars/:owner/:repo by serving the
// repository's releases feed, falling back to plain tags for repos that
// do not publish release notes.
func GitHubStarsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)

	return gitHubStarsFeed(
		c,
		fmt.Sprintf("%s/repos/%s/%s/releases?per_page=%d", gitHubAPIBase, owner, repo, limit),
		fmt.Sprintf("%s/repos/%s/%s/tags", gitHubAPIBase, owner, repo),
		owner,
		repo,
	)
}

// gitHubStarsFeed builds the stars feed from explicit API URLs.
func gitHubStarsFeed(c *ctxpkg.Context, releasesURL, tagsURL, owner, repo string) (*models.Feed, error) {
	ctx := c.Parent()
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)

	var releases []GitHubRelease
	if err := gitHubGetJSON(ctx, c.Client(), releasesURL, &releases); err != nil {
		return nil, err
	}

	if len(releases) > 0 {
		feed := routeutils.NewFeed(
			fmt.Sprintf("%s/%s Releases", owner, repo),
			fmt.Sprintf("%s/%s/%s/releases", gitHubWebBase, owner, repo),
			fmt.Sprintf("Latest releases from %s/%s", owner, repo),
		)
		routeutils.AppendMappedItems(feed, releases, limit, func(release GitHubRelease) *models.Item {
			tag := release.TagName
			link := release.HTMLURL
			if tag == "" {
				return nil
			}
			if link == "" {
				link = fmt.Sprintf("%s/%s/%s/releases/tag/%s", gitHubWebBase, owner, repo, tag)
			}
			item := routeutils.NewItem("Release "+tag, link, excerptBody(release.Body, 600), release.PublishedAt)
			item.GUID = fmt.Sprintf("gh-release-%s", tag)
			return item
		})
		return feed, nil
	}

	// No releases (or empty body): fall back to plain tags.
	var tags []GitHubTag
	if err := gitHubGetJSON(ctx, c.Client(), tagsURL, &tags); err != nil {
		return nil, err
	}
	feed := routeutils.NewFeed(
		fmt.Sprintf("%s/%s Tags", owner, repo),
		fmt.Sprintf("%s/%s/%s/tags", gitHubWebBase, owner, repo),
		fmt.Sprintf("Latest tags from %s/%s (no published releases)", owner, repo),
	)
	routeutils.AppendMappedItems(feed, tags, limit, func(tag GitHubTag) *models.Item {
		if tag.Name == "" {
			return nil
		}
		link := fmt.Sprintf("%s/%s/%s/tree/%s", gitHubWebBase, owner, repo, tag.Name)
		item := routeutils.NewItem("Tag "+tag.Name, link, "", time.Time{})
		item.GUID = fmt.Sprintf("gh-release-%s", tag.Name)
		return item
	})
	return feed, nil
}

// GitHubTag is a stripped-down entry of the /repos/{owner}/{repo}/tags endpoint.
type GitHubTag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
}
