package main

import (
	"encoding/json"
	"time"

	ctxpkg "github.com/rsshub/go/pkg/context"
	"github.com/rsshub/go/pkg/models"
	"github.com/rsshub/go/pkg/registry"
)

// GitHub repos route
func init() {
	route := &models.Route{
		Path:         "/github/repos/:owner/:repo",
		Name:         "GitHub Repository Releases",
		Example:      "github/repos/DIYgod/RSSHub",
		Maintainers:  []string{"yourname"},
		Description:  "Fetch releases from a GitHub repository",
		Categories:   []models.Category{{Name: "dev"}},
		Features:     models.Features{SupportRadar: true},
		Handler:      GitHubReposHandler,
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Repository owner"},
			{Name: "repo", Required: true, Description: "Repository name"},
		},
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// GitHubReposHandler handles /github/repos/:owner/:repo
func GitHubReposHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")

	url := "https://api.github.com/repos/" + owner + "/" + repo + "/releases"
	ctx := c.Parent()

	resp, err := c.Client().Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var releases []GitHubRelease
	if err := json.Unmarshal(resp, &releases); err != nil {
		return nil, err
	}

	feed := &models.Feed{
		Title:       owner + "/" + repo + " Releases",
		Link:        "https://github.com/" + owner + "/" + repo + "/releases",
		Description: "Latest releases from " + owner + "/" + repo,
	}

	for _, release := range releases {
		if release.TagName != "" {
			item := models.Item{
				Title:       "Release " + release.TagName,
				Link:        release.HTMLURL,
				GUID:        release.HTMLURL,
				Description: release.Body,
				PubDate:     release.PublishedAt,
			}
			feed.Items = append(feed.Items, item)
		}
	}

	return feed, nil
}

type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
}
