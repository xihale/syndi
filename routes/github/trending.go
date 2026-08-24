package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitHubTrendingRoute = routeutils.RouteSpec{
	Path:        "trending/:language",
	Name:        "GitHub Trending",
	Example:     "github/trending/go",
	Maintainers: []string{"xihale"},
	Description: "Fetch trending repositories on GitHub by language",
	Categories:  []models.Category{{Name: "dev"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("language", "Programming language (use 'all' for any language)"),
	},
	CacheTTL: 30 * time.Minute, // Trending changes moderately
	Handler:  GitHubTrendingHandler,
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
		item := parseGitHubTrendingItem(sel, time.Now())
		routeutils.AddItem(feed, item)
	})

	return feed, nil
}

func parseGitHubTrendingItem(sel *parser.Selection, now time.Time) *models.Item {
	if sel == nil {
		return nil
	}

	titleSel := sel.Find("h2 a")
	title := normalizeSpace(titleSel.Text())

	href, _ := titleSel.Attr("href")
	if href == "" || title == "" {
		return nil
	}
	link := "https://github.com" + href

	item := routeutils.NewItem(
		title,
		link,
		buildTrendingDescription(sel),
		now,
	)
	item.GUID = link

	languageName := normalizeSpace(firstSelectionText(sel, "span[itemprop='programmingLanguage']"))
	if languageName != "" {
		routeutils.SetCategories(item, languageName)
	}

	starsToday := extractStarsToday(sel)
	if starsToday != "" {
		routeutils.SetCategories(item, starsToday)
	}

	return item
}

func buildTrendingDescription(sel *parser.Selection) string {
	description := normalizeSpace(firstSelectionText(sel, "p"))
	stats := extractTrendingStats(sel)
	if len(stats) == 0 {
		return description
	}
	if description == "" {
		return "Stats: " + strings.Join(stats, " | ")
	}
	return description + "<br/>Stats: " + strings.Join(stats, " | ")
}

func extractTrendingStats(sel *parser.Selection) []string {
	stats := make([]string, 0, 2)
	sel.Find("a[href*='/stargazers'],a[href*='/forks']").Each(func(i int, statSel *parser.Selection) {
		text := normalizeSpace(statSel.Text())
		if text != "" {
			stats = append(stats, text)
		}
	})
	return stats
}

func extractStarsToday(sel *parser.Selection) string {
	starsToday := ""
	sel.Find("span").Each(func(i int, spanSel *parser.Selection) {
		text := normalizeSpace(spanSel.Text())
		if strings.Contains(text, "stars today") {
			starsToday = text
		}
	})
	return starsToday
}

func firstSelectionText(sel *parser.Selection, selector string) string {
	if sel == nil || sel.Selection == nil {
		return ""
	}
	return sel.Selection.Find(selector).First().Text()
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}
