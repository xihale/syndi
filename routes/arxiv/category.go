package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var arxivCategoryRoute = routeutils.RouteSpec{
	Path:        "category/:category",
	Name:        "arXiv Category New Papers",
	Example:     "arxiv/category/cs.AI",
	Maintainers: []string{"xihale"},
	Description: "Latest papers submitted to an arXiv category (e.g. cs.AI, math.AG, physics.comp-ph)",
	Categories:  []models.Category{{Name: "study"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("category", "arXiv category, e.g. cs.AI"),
		routeutils.OptionalParam("limit", "Number of papers, default 20, max 100"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  ArxivCategoryHandler,
}

// ArxivCategoryHandler handles /arxiv/category/:category
func ArxivCategoryHandler(c *ctxpkg.Context) (*models.Feed, error) {
	category := strings.TrimSpace(c.Param("category"))
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 100)
	ctx := c.Parent()

	url := fmt.Sprintf("https://export.arxiv.org/api/query?search_query=cat:%s&sortBy=submittedDate&sortOrder=descending&max_results=%d", category, limit)

	var resp arxivAtom
	if err := routeutils.GetXML(ctx, c.Client(), url, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("arXiv %s", category),
		fmt.Sprintf("https://arxiv.org/list/%s/recent", category),
		fmt.Sprintf("New submissions to arXiv category %s", category),
	)

	for _, e := range resp.Entries {
		link := e.AlternateLink()
		if link == "" {
			continue
		}
		pub := e.Published
		if pub.IsZero() {
			pub = e.Updated
		}
		item := routeutils.NewItem(
			strings.TrimSpace(e.Title),
			link,
			buildArxivDescription(e),
			pub,
		)
		item.GUID = e.ID
		if len(e.Authors) > 0 {
			routeutils.SetItemAuthor(item, e.Authors[0].Name, "", "")
			if len(e.Authors) > 1 {
				names := make([]string, 0, len(e.Authors))
				for _, a := range e.Authors {
					names = append(names, a.Name)
				}
				item.Description += "<br/><em>Authors:</em> " + strings.Join(names, ", ")
			}
		}
		routeutils.SetCategories(item, category)
		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

func buildArxivDescription(e arxivEntry) string {
	var b strings.Builder
	b.WriteString("<p>")
	b.WriteString(strings.TrimSpace(e.Summary))
	b.WriteString("</p>")
	if comment := strings.TrimSpace(e.Comment); comment != "" {
		b.WriteString("<p><em>Comments:</em> " + comment + "</p>")
	}
	return b.String()
}

type arxivAtom struct {
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	ID        string       `xml:"id"`
	Title     string       `xml:"title"`
	Summary   string       `xml:"summary"`
	Comment   string       `xml:"http://arxiv.org/schemas/atom comment"`
	Published time.Time    `xml:"published"`
	Updated   time.Time    `xml:"updated"`
	Authors   []arxivAutho `xml:"author"`
	Links     []arxivLink  `xml:"link"`
}

func (e *arxivEntry) AlternateLink() string {
	for _, l := range e.Links {
		if l.Rel == "" || l.Rel == "alternate" {
			return l.Href
		}
	}
	if len(e.Links) > 0 {
		return e.Links[0].Href
	}
	return ""
}

type arxivAutho struct {
	Name string `xml:"name"`
}

type arxivLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}
