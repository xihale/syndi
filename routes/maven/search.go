package routes

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var mavenSearchRoute = routeutils.RouteSpec{
	Path:        "search/:query",
	Name:        "Maven Central Search",
	Example:     "maven/search/guava",
	Maintainers: []string{"xihale"},
	Description: "Search artifacts on Maven Central",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("query", "Solr query, e.g. g:com.google.guava or a:guava"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  MavenSearchHandler,
}

// MavenSearchHandler handles /maven/search/:query
func MavenSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	query := c.Param("query")
	ctx := c.Parent()

	apiURL := fmt.Sprintf("https://search.maven.org/solrsearch/select?q=%s&rows=20&wt=json", url.QueryEscape(query))

	var response MavenSearchResponse
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &response); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Maven Central search: %s", query),
		fmt.Sprintf("https://central.sonatype.com/search?q=%s", url.QueryEscape(query)),
		fmt.Sprintf("Maven Central artifacts matching %s", query),
	)
	routeutils.AppendMappedItems(feed, response.Response.Docs, 20, func(doc MavenDoc) *models.Item {
		group, artifact := splitMavenID(doc.ID)
		if artifact == "" {
			return nil
		}
		link := fmt.Sprintf("https://central.sonatype.com/artifact/%s/%s/", group, artifact)

		item := routeutils.NewItem(
			fmt.Sprintf("%s : %s", doc.ID, doc.LatestVersion),
			link,
			"",
			time.UnixMilli(doc.Timestamp),
		)
		item.GUID = link
		routeutils.SetCategories(item, group)
		return item
	})

	return feed, nil
}

// splitMavenID splits "group:artifact" at the last colon.
func splitMavenID(id string) (string, string) {
	if i := strings.LastIndexByte(id, ':'); i >= 0 {
		return id[:i], id[i+1:]
	}
	return "", id
}

type MavenSearchResponse struct {
	Response struct {
		NumFound int        `json:"numFound"`
		Docs     []MavenDoc `json:"docs"`
	} `json:"response"`
}

type MavenDoc struct {
	ID            string `json:"id"`
	LatestVersion string `json:"latestVersion"`
	Timestamp     int64  `json:"timestamp"`
}
