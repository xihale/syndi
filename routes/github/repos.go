package routes

import (
	"context"
	"fmt"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
)

// GitHub repos route
func init() {
	cacheTTL := 1 * time.Hour // GitHub releases are infrequent

	route := &models.Route{
		Path:        "/github/repos/:owner/:repo",
		Name:        "GitHub Repository Releases",
		Example:     "github/repos/DIYgod/RSSHub",
		Maintainers: []string{"yourname"},
		Description: "Fetch releases from a GitHub repository",
		Categories:  []models.Category{{Name: "dev"}},
		Features:    models.Features{SupportRadar: true},
		Handler:     GitHubReposHandler,
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Repository owner"},
			{Name: "repo", Required: true, Description: "Repository name"},
		},
		CacheTTL: &cacheTTL,
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// GitHubReposHandler handles /github/repos/:owner/:repo
func GitHubReposHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	limit := parsePositiveLimit(c.QueryParam("limit"), 20, 100)
	includePrerelease := parseBoolDefault(c.QueryParam("include_prerelease"), true)

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
	feed.Items = make([]models.Item, 0, limit)

	for _, release := range releases {
		if len(feed.Items) >= limit {
			break
		}
		if release.TagName == "" || release.HTMLURL == "" {
			continue
		}
		if !includePrerelease && release.Prerelease {
			continue
		}

		item := models.Item{
			Title:       "Release " + release.TagName,
			Link:        release.HTMLURL,
			GUID:        release.HTMLURL,
			Description: release.Body,
			PubDate:     release.PublishedAt,
		}
		feed.Items = append(feed.Items, item)
	}

	return feed, nil
}

type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Prerelease  bool      `json:"prerelease"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
}
