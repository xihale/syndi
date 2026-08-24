package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

const smashingBaseURL = "https://www.smashingmagazine.com"

var latestRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Latest Articles",
	Example:     "smashingmagazine",
	Maintainers: []string{"xihale"},
	Description: "Latest articles on Smashing Magazine (native RSS, normalized)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     LatestHandler,
}

// LatestHandler handles /smashingmagazine
func LatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), smashingBaseURL+"/feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "Smashing Magazine"
	feed.Link = smashingBaseURL + "/"
	feed.Description = "Latest articles on Smashing Magazine for web designers and developers"
	return feed, nil
}

var categoryRoute = routeutils.RouteSpec{
	Path:        ":category",
	Name:        "Category Articles",
	Example:     "smashingmagazine/react",
	Maintainers: []string{"xihale"},
	Description: "Articles from a Smashing Magazine category page, e.g. css, javascript, design, react, ux",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("category", "Category slug from the site URL, e.g. css or web-design"),
	},
	CacheTTL: 3 * time.Hour,
	Handler:  CategoryHandler,
}

// CategoryHandler handles /smashingmagazine/:category
func CategoryHandler(c *ctxpkg.Context) (*models.Feed, error) {
	category := strings.TrimSpace(c.Param("category"))
	if category == "" || strings.ContainsAny(category, "/?&#") {
		return nil, fmt.Errorf("smashingmagazine: invalid category %q", category)
	}
	pageURL := fmt.Sprintf("%s/category/%s/", smashingBaseURL, category)

	doc, err := routeutils.GetHTML(c.Parent(), c.Client(), pageURL)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Smashing Magazine - %s", category),
		pageURL,
		fmt.Sprintf("Smashing Magazine articles in the %s category", category),
	)

	doc.Each("article.article--post", func(_ int, card *parser.Selection) {
		a := card.Find("h2.article--post__title a")
		if a == nil || a.Length() == 0 {
			return
		}
		title := strings.TrimSpace(a.Text())
		href := a.AttrOr("href", "")
		if title == "" || href == "" {
			return
		}
		link := href
		if !strings.HasPrefix(link, "http") {
			link = smashingBaseURL + link
		}

		description := ""
		var pubDate time.Time
		teaser := card.Find("p.article--post__teaser")
		if teaser != nil && teaser.Length() > 0 {
			var pubDateStr string
			if tm := teaser.Find("time"); tm != nil && tm.Length() > 0 {
				pubDateStr = tm.AttrOr("datetime", "")
				tm.Remove()
			}
			if more := teaser.Find("a.read-more-link"); more != nil && more.Length() > 0 {
				more.Remove()
			}
			description = html.EscapeString(strings.TrimSpace(teaser.Text()))

			if pubDateStr != "" {
				if t, err := dateutil.ParseDate(pubDateStr); err == nil {
					pubDate = t
				}
			}
		}

		item := routeutils.NewItem(title, link, description, pubDate)
		if author := card.Find("span.article--post__author-name a"); author != nil && author.Length() > 0 {
			routeutils.SetItemAuthor(item, strings.TrimSpace(author.Text()), "", "")
		}
		routeutils.AddItem(feed, item)
	})

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("smashingmagazine: no articles found for category %q", category)
	}
	return feed, nil
}
