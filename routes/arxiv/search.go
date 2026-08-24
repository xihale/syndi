package routes

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var arxivSearchRoute = routeutils.RouteSpec{
	Path:        "search/:keyword",
	Name:        "arXiv Search",
	Example:     "arxiv/search/federated+learning",
	Maintainers: []string{"xihale"},
	Description: "Search arXiv papers by keyword (all fields)",
	Categories:  []models.Category{{Name: "study"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("keyword", "Search keyword (URL-encoded)"),
		routeutils.OptionalParam("limit", "Number of results, default 20, max 100"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  ArxivSearchHandler,
}

// ArxivSearchHandler handles /arxiv/search/:keyword
func ArxivSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	keyword := c.Param("keyword")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 100)
	ctx := c.Parent()

	url := "https://export.arxiv.org/api/query?search_query=all:" + url.QueryEscape(keyword) +
		"&sortBy=relevance&sortOrder=descending&max_results=" + itoa(limit)

	var resp arxivAtom
	if err := routeutils.GetXML(ctx, c.Client(), url, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"arXiv Search: "+keyword,
		"https://arxiv.org/abs/"+keyword,
		"arXiv search results for "+keyword,
	)
	feed.Link = "https://arxiv.org/list/cs/recent"

	for _, e := range resp.Entries {
		link := e.AlternateLink()
		if link == "" {
			continue
		}
		pub := e.Published
		if pub.IsZero() {
			pub = e.Updated
		}
		item := routeutils.NewItem(strings.TrimSpace(e.Title), link, buildArxivDescription(e), pub)
		item.GUID = e.ID
		if len(e.Authors) > 0 {
			routeutils.SetItemAuthor(item, e.Authors[0].Name, "", "")
		}
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
