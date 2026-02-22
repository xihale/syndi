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

// GitHub user repositories route
func init() {
	cacheTTL := 2 * time.Hour // User repositories change slowly

	route := &models.Route{
		Path:        "/github/users/:username/repos",
		Name:        "GitHub User Repositories",
		Example:     "github/users/torvalds/repos",
		Maintainers: []string{"yourname"},
		Description: "Fetches all public repositories for a GitHub user",
		Categories:  []models.Category{{Name: "dev"}},
		Features:    models.Features{SupportRadar: true},
		Handler:     GitHubUserReposHandler,
		Parameters: []models.Parameter{
			{Name: "username", Required: true, Description: "GitHub username"},
		},
		CacheTTL: &cacheTTL,
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// GitHubUserReposHandler handles /github/users/:username/repos
func GitHubUserReposHandler(c *ctxpkg.Context) (*models.Feed, error) {
	username := c.Param("username")
	limit := parsePositiveLimit(c.QueryParam("limit"), 30, 100)

	// Build GitHub API URL - fetch public repos sorted by updated desc
	url := fmt.Sprintf(
		"https://api.github.com/users/%s/repos?type=public&sort=updated&direction=desc&per_page=%d",
		username,
		limit,
	)

	ctx, cancel := context.WithTimeout(c.Parent(), 12*time.Second)
	defer cancel()

	var repos []GitHubRepo
	if err := routeutils.GetJSON(ctx, c.Client(), url, &repos); err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub API: %w", err)
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s's Repositories", username),
		fmt.Sprintf("https://github.com/%s?tab=repos", username),
		fmt.Sprintf("Public repositories for GitHub user %s", username),
	)
	feed.Items = make([]models.Item, 0, limit)

	for _, repo := range repos {
		if len(feed.Items) >= limit {
			break
		}
		if repo.FullName == "" || repo.HTMLURL == "" {
			continue
		}

		item := models.Item{
			Title:       repo.FullName,
			Link:        repo.HTMLURL,
			GUID:        repo.HTMLURL,
			Description: repo.Description,
			PubDate:     repo.UpdatedAt,
			Updated:     &repo.UpdatedAt,
		}
		if repo.Language != "" {
			item.Categories = []string{repo.Language}
		}

		// Add author info
		if repo.Owner.Login != "" {
			item.Author = &models.Author{
				Name: repo.Owner.Login,
				URI:  repo.Owner.HTMLURL,
			}
		}

		feed.Items = append(feed.Items, item)
	}

	return feed, nil
}

type GitHubRepo struct {
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	FullName    string      `json:"full_name"`
	Description string      `json:"description"`
	HTMLURL     string      `json:"html_url"`
	Language    string      `json:"language"`
	Stargazers  int         `json:"stargazers_count"`
	Watchers    int         `json:"watchers_count"`
	Forks       int         `json:"forks_count"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	PushedAt    time.Time   `json:"pushed_at"`
	Owner       GitHubOwner `json:"owner"`
}

type GitHubOwner struct {
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
	Type    string `json:"type"`
}
