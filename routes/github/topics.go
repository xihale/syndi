package routes

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitHubTopicsRoute = routeutils.RouteSpec{
	Path:        "topics/:name",
	Name:        "GitHub Topic Repositories",
	Example:     "github/topics/framework",
	Maintainers: []string{"xihale"},
	Description: "Fetch repositories of a GitHub topic, sorted by stars",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("name", "Topic name, e.g. framework (see github.com/topics)"),
		routeutils.OptionalParam("qs", "Query string like l=go&s=stars&o=desc (l: language, s: sort by stars/forks/updated, o: asc/desc)"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitHubTopicsHandler,
}

var gitHubTopicsQSRoute = routeutils.RouteSpec{
	Path:        "topics/:name/:qs",
	Name:        "GitHub Topic Repositories with Filters",
	Example:     "github/topics/framework/l=go&s=stars",
	Maintainers: []string{"xihale"},
	Description: "Fetch repositories of a GitHub topic filtered through a query string (l=language, s=sort criteria, o=order)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("name", "Topic name, e.g. framework (see github.com/topics)"),
		routeutils.RequiredParam("qs", "Query string like l=php&o=desc&s=stars"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitHubTopicsHandler,
}

// GitHubTopicsHandler handles /github/topics/:name[/:qs].
// The upstream HTML page is fragile to parse, so this route uses the
// repository search API instead and maps the well-known qs keys onto it.
func GitHubTopicsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" || strings.ContainsAny(name, "/ ") {
		return nil, fmt.Errorf("invalid topic name %q", c.Param("name"))
	}
	qs := normalizeTopicQS(strings.TrimPrefix(c.Param("qs"), "?"))
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)

	query := "topic:" + name
	sortBy := "stars"
	order := "desc"

	for key, value := range qs {
		switch key {
		case "l":
			if value != "" {
				query += " language:" + value
			}
		case "s":
			if allowed := routeutils.ParseEnum(value, "", "stars", "forks", "updated"); allowed != "" {
				sortBy = allowed
			}
		case "o":
			order = routeutils.ParseEnum(value, "desc", "asc", "desc")
		}
	}

	endpoint := fmt.Sprintf("%s/search/repositories?q=%s&sort=%s&order=%s&per_page=%d",
		gitHubAPIBase, url.QueryEscape(query), sortBy, order, limit)
	ctx := c.Parent()

	var result GitHubRepoSearch
	if err := gitHubGetJSON(ctx, c.Client(), endpoint, &result); err != nil {
		return nil, err
	}

	title := fmt.Sprintf("GitHub Topic: %s", name)
	link := fmt.Sprintf("%s/topics/%s", gitHubWebBase, name)
	feedDescription := fmt.Sprintf("Repositories under the %q topic sorted by %s", name, sortBy)
	if _, hasLang := qs["l"]; hasLang && qs["l"] != "" {
		title += fmt.Sprintf(" (%s)", qs["l"])
	}

	feed := routeutils.NewFeed(title, link, feedDescription)
	routeutils.AppendMappedItems(feed, result.Items, limit, func(repo GitHubSearchRepo) *models.Item {
		if repo.FullName == "" || repo.HTMLURL == "" {
			return nil
		}

		pubDate := repo.PushedAt
		if pubDate.IsZero() {
			pubDate = repo.UpdatedAt
		}

		description := ""
		if repo.Description != "" {
			description = html.EscapeString(repo.Description)
		}
		description += fmt.Sprintf("<br/>Stars: %d | Forks: %d", repo.StargazersCount, repo.ForksCount)

		item := routeutils.NewItem(repo.FullName, repo.HTMLURL, description, pubDate)
		item.GUID = repo.HTMLURL
		if repo.Owner.Login != "" {
			routeutils.SetItemAuthor(item, repo.Owner.Login, "", fmt.Sprintf("%s/%s", gitHubWebBase, repo.Owner.Login))
		}
		if repo.Language != "" {
			routeutils.SetCategories(item, repo.Language)
		}
		for i, topic := range repo.Topics {
			if i >= 5 {
				break
			}
			routeutils.SetCategories(item, topic)
		}
		return item
	})

	return feed, nil
}

// normalizeTopicQS splits an inline query-string path segment into pairs,
// keeping only the last occurrence of each key with trimmed values.
func normalizeTopicQS(raw string) map[string]string {
	pairs := make(map[string]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return pairs
	}
	for _, pair := range strings.Split(raw, "&") {
		key, value, found := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !found || key == "" {
			continue
		}
		pairs[key] = value
	}
	return pairs
}

type GitHubRepoSearch struct {
	TotalCount int                `json:"total_count"`
	Items      []GitHubSearchRepo `json:"items"`
}

type GitHubSearchRepo struct {
	FullName        string    `json:"full_name"`
	HTMLURL         string    `json:"html_url"`
	Description     string    `json:"description"`
	Language        string    `json:"language"`
	StargazersCount int       `json:"stargazers_count"`
	ForksCount      int       `json:"forks_count"`
	PushedAt        time.Time `json:"pushed_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Owner           struct {
		Login   string `json:"login"`
		HTMLURL string `json:"html_url"`
	} `json:"owner"`
	Topics []string `json:"topics"`
}
