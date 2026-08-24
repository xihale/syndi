package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var dockerHubTagsRoute = routeutils.RouteSpec{
	Path:        "tags/*repo",
	Name:        "Docker Hub Image Tags",
	Example:     "dockerhub/tags/library/nginx",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest tags pushed to a Docker Hub image",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("repo", "Image repository including namespace, e.g. library/nginx"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  DockerHubTagsHandler,
}

// DockerHubTagsHandler handles /dockerhub/tags/*repo
func DockerHubTagsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	repo := strings.Trim(c.Param("repo"), "/")
	if repo == "" {
		return nil, fmt.Errorf("repo parameter is required, e.g. library/nginx")
	}
	ctx := c.Parent()

	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags?page_size=25", repo)

	var response DockerHubTagsResponse
	if err := routeutils.GetJSON(ctx, c.Client(), url, &response); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s Tags - Docker Hub", repo),
		fmt.Sprintf("https://hub.docker.com/r/%s/tags", repo),
		fmt.Sprintf("Latest tags of Docker Hub image %s", repo),
	)
	routeutils.AppendMappedItems(feed, response.Results, 25, func(tag DockerHubTag) *models.Item {
		if tag.Name == "" {
			return nil
		}
		link := fmt.Sprintf("https://hub.docker.com/r/%s/tags?name=%s", repo, tag.Name)

		var desc strings.Builder
		if tag.Digest != "" {
			short := strings.TrimPrefix(tag.Digest, "sha256:")
			if len(short) > 12 {
				short = short[:12]
			}
			fmt.Fprintf(&desc, "Digest: <code>sha256:%s</code>", html.EscapeString(short))
		}
		if len(tag.Images) > 0 {
			if desc.Len() > 0 {
				desc.WriteString("<br/>")
			}
			fmt.Fprintf(&desc, "%d platform images", len(tag.Images))
		}

		item := routeutils.NewItem(
			fmt.Sprintf("%s:%s", repo, tag.Name),
			link,
			desc.String(),
			tag.TagLastPushed,
		)
		item.GUID = fmt.Sprintf("dockerhub-%s-%s", repo, tag.Name)
		routeutils.SetCategories(item, tag.Name)
		return item
	})

	return feed, nil
}

type DockerHubTagsResponse struct {
	Results []DockerHubTag `json:"results"`
}

type DockerHubTag struct {
	Name          string     `json:"name"`
	Digest        string     `json:"digest"`
	Images        []struct{} `json:"images"`
	TagLastPushed time.Time  `json:"tag_last_pushed"`
}
