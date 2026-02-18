package routes

import (
	"encoding/json"
	"fmt"
	"time"

	ctxpkg "github.com/rsshub/go/pkg/context"
	"github.com/rsshub/go/pkg/logger"
	"github.com/rsshub/go/pkg/models"
	"github.com/rsshub/go/pkg/registry"
	"go.uber.org/zap"
)

// safeSlice returns a slice of up to max elements from the slice
func safeSlice(slice []byte, max int) []byte {
	if len(slice) > max {
		return slice[:max]
	}
	return slice
}

// GitHub user repositories route
func init() {
	route := &models.Route{
		Path:         "/github/users/:username/repos",
		Name:         "GitHub User Repositories",
		Example:      "github/users/torvalds/repos",
		Maintainers:  []string{"yourname"},
		Description:  "Fetches all public repositories for a GitHub user",
		Categories:   []models.Category{{Name: "dev"}},
		Features:     models.Features{SupportRadar: true},
		Handler:      GitHubUserReposHandler,
		Parameters: []models.Parameter{
			{Name: "username", Required: true, Description: "GitHub username"},
		},
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// GitHubUserReposHandler handles /github/users/:username/repos
func GitHubUserReposHandler(c *ctxpkg.Context) (*models.Feed, error) {
	username := c.Param("username")
	logger.Info("GitHubUserReposHandler called", zap.String("username", username))

	// Build GitHub API URL - fetch public repos sorted by updated desc
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?type=public&sort=updated&direction=desc",
		username)
	logger.Info("Fetching from GitHub", zap.String("url", url))

	ctx := c.Parent()

	resp, err := c.Client().Get(ctx, url)
	if err != nil {
		logger.Error("Failed to fetch from GitHub", zap.Error(err), zap.String("url", url))
		return nil, fmt.Errorf("failed to fetch from GitHub API: %w", err)
	}
	logger.Info("Received response", zap.Int("response_length", len(resp)))

	var repos []GitHubRepo
	if err := json.Unmarshal(resp, &repos); err != nil {
		logger.Error("Failed to parse response", zap.Error(err), zap.String("response", string(safeSlice(resp, 500))))
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}
	logger.Info("Parsed repos", zap.Int("count", len(repos)))

	feed := &models.Feed{
		Title:       fmt.Sprintf("%s's Repositories", username),
		Link:        fmt.Sprintf("https://github.com/%s?tab=repos", username),
		Description: fmt.Sprintf("Public repositories for GitHub user %s", username),
	}

	for _, repo := range repos {
		item := models.Item{
			Title:       repo.FullName,
			Link:        repo.HTMLURL,
			GUID:        repo.HTMLURL,
			Description: repo.Description,
			PubDate:     repo.UpdatedAt,
			Categories:  []string{repo.Language},
			Updated:     &repo.UpdatedAt,
		}

		// Add author info
		if username != "" {
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
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	HTMLURL     string    `json:"html_url"`
	Language    string    `json:"language"`
	Stargazers  int       `json:"stargazers_count"`
	Watchers    int       `json:"watchers_count"`
	Forks       int       `json:"forks_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PushedAt    time.Time `json:"pushed_at"`
	Owner       GitHubOwner `json:"owner"`
}

type GitHubOwner struct {
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
	Type    string `json:"type"`
}
