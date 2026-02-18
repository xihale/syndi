package routes

import (
	"fmt"
	"time"

	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
	"github.com/xihale/rsshub-go/internal/parser"
	"github.com/xihale/rsshub-go/internal/routeutils"
)

func init() {
	cacheTTL := 30 * time.Minute // Trending changes moderately

	route := &models.Route{
		Path:         "/github/trending/:language",
		Name:         "GitHub Trending",
		Example:      "github/trending/go",
		Maintainers:  []string{"yourname"},
		Description:  "Fetch trending repositories on GitHub by language",
		Categories:   []models.Category{{Name: "dev"}},
		Features:     models.Features{},
		Handler:      GitHubTrendingHandler,
		Parameters: []models.Parameter{
			{Name: "language", Required: false, Description: "Programming language (use 'all' for any language)"},
		},
		CacheTTL: &cacheTTL,
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// GitHubTrendingHandler handles /github/trending/:language
func GitHubTrendingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	language := c.Param("language")
	if language == "" {
		language = "all"
	}

	ctx := c.Parent()

	// GitHub doesn't have an official API for trending, so we use the HTML page
	url := fmt.Sprintf("https://github.com/trending/%s", language)

	doc, err := routeutils.GetHTML(ctx, c.Client(), url)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("GitHub Trending - %s", language),
		url,
		fmt.Sprintf("Trending repositories written in %s", language),
	)

	// Select repo articles
	doc.Each("article.Box-row", func(i int, sel *parser.Selection) {
		// Get title
		titleSel := sel.Find("h2 a")
		title := titleSel.Text()

		// Get link
		href, _ := titleSel.Attr("href")
		link := "https://github.com" + href

		// Get description
		desc := sel.Find("p").First().Text()

		// Get stars, forks, stars today
		stats := make([]string, 0)
		sel.Find("a[dhref]").Each(func(i int, statSel *parser.Selection) {
			text := statSel.Text()
			if text != "" {
				stats = append(stats, text)
			}
		})

		description := desc
		if len(stats) > 0 {
			description += "<br/>" + fmt.Sprintf("Stats: %s", stats)
		}

		// Get programming language
		languageSpan := sel.Find("span[itemprop='programmingLanguage']").First()
		languageName := languageSpan.Text()

		// Get stars today from the special span
		starsToday := sel.Find("span.d-inline-block+ span").Text()

		item := routeutils.NewItem(
			title,
			link,
			description,
			time.Now(),
		)
		item.GUID = link

		// Set categories
		if languageName != "" {
			routeutils.SetCategories(item, languageName)
		}
		if starsToday != "" {
			routeutils.SetCategories(item, starsToday+" today")
		}

		routeutils.AddItem(feed, item)
	})

	return feed, nil
}
