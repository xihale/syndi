package routes

import (
	"fmt"
	"html"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitLabExploreRoute = routeutils.RouteSpec{
	Path:        "explore/:sort",
	Name:        "GitLab Explore Projects",
	Example:     "gitlab/explore/last_activity_at",
	Maintainers: []string{"xihale"},
	Description: "Explore trending projects on GitLab sorted by activity, creation date or stars",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("sort", "Sort key: last_activity_at | created_at | stars"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  GitLabExploreHandler,
}

// GitLabExploreHandler handles /gitlab/explore/:sort
func GitLabExploreHandler(c *ctxpkg.Context) (*models.Feed, error) {
	sortKey := routeutils.ParseEnum(c.Param("sort"), "last_activity_at", "last_activity_at", "created_at", "stars")

	// The API has no "stars" order_by value; star_count is the equivalent.
	orderBy := sortKey
	if orderBy == "stars" {
		orderBy = "star_count"
	}

	ctx := c.Parent()

	url := fmt.Sprintf(
		"https://gitlab.com/api/v4/projects?order_by=%s&sort=desc&per_page=20&simple=true",
		orderBy,
	)

	var projects []GitLabProject
	if err := routeutils.GetJSON(ctx, c.Client(), url, &projects); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("GitLab Explore (%s)", sortKey),
		fmt.Sprintf("https://gitlab.com/explore?sort=%s", orderBy),
		fmt.Sprintf("Popular GitLab projects by %s", sortKey),
	)
	routeutils.AppendMappedItems(feed, projects, 20, func(project GitLabProject) *models.Item {
		if project.WebURL == "" || project.Name == "" {
			return nil
		}

		description := html.EscapeString(project.Description)

		item := routeutils.NewItem(project.Name, project.WebURL, description, project.LastActivityAt)
		item.GUID = project.WebURL
		routeutils.SetCategories(item, fmt.Sprintf("stars:%d", project.StarCount))
		return item
	})

	return feed, nil
}
