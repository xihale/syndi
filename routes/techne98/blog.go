package routes

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xihale/rsshub-go/internal/parser"
	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/utils/date"
)

var techne98BlogRoute = routeutils.RouteSpec{
	Path:        "blog",
	Name:        "techne98 - blog",
	Example:     "techne98/blog",
	Maintainers: []string{"xihale"},
	Description: "Fetch blog posts from techne98.com",
	Categories:  []models.Category{{Name: "blog"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Maximum number of articles to return (default: all)"),
	},
	CacheTTL: 5 * 24 * time.Hour, // Blog list cache: 5 days
	Handler:  Techne98BlogHandler,
}

// Techne98BlogHandler handles /techne98/blog
func Techne98BlogHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()
	rootURL := "https://techne98.com"
	currentURL := rootURL + "/blog/"

	doc, err := routeutils.GetHTML(ctx, c.Client(), currentURL)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"techne98 - blog",
		currentURL,
		"Blog posts from techne98.com",
	)

	maxItems := routeutils.ParseOptionalPositiveInt(c.QueryParam("limit"))
	items := make([]*models.Item, 0)

	doc.Each("main .grid a.group", func(i int, sel *parser.Selection) {
		if maxItems != nil && len(items) >= *maxItems {
			return
		}

		title := strings.TrimSpace(sel.Find("h2").Text())

		href, exists := sel.Attr("href")
		if !exists || href == "" {
			return
		}

		link := href
		if strings.HasPrefix(href, "/") {
			link = rootURL + href
		}

		pubDateText := strings.TrimSpace(sel.Find("span").Text())
		pubDate := time.Now()
		if pubDateText != "" {
			if parsed, err := date.ParseDate(pubDateText); err == nil && !parsed.IsZero() {
				pubDate = parsed
			}
		}

		item := routeutils.NewItem(
			title,
			link,
			"",
			pubDate,
		)
		item.GUID = link

		items = append(items, item)
	})

	populateTechne98Descriptions(ctx, c, rootURL, items)

	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	for _, item := range items {
		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

func populateTechne98Descriptions(ctx context.Context, c *ctxpkg.Context, rootURL string, items []*models.Item) {
	if len(items) == 0 {
		return
	}

	workerCount := 4
	if len(items) < workerCount {
		workerCount = len(items)
	}

	jobs := make(chan *models.Item)
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				description, err := fetchTechne98Description(ctx, c, rootURL, item.Link)
				if err != nil || description == "" {
					continue
				}
				item.Description = description
			}
		}()
	}

enqueue:
	for _, item := range items {
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- item:
		}
	}
	close(jobs)
	wg.Wait()
}

func fetchTechne98Description(ctx context.Context, c *ctxpkg.Context, rootURL, link string) (string, error) {
	contentCacheKey := fmt.Sprintf("techne98:article:%s", link)
	cachedValue, err := c.CacheTryGet(contentCacheKey, 30*24*time.Hour, func() (interface{}, error) {
		detailDoc, err := routeutils.GetHTML(ctx, c.Client(), link)
		if err != nil {
			return "", err
		}
		articleSel := detailDoc.Find("article.prose-content")
		if articleSel.Length() == 0 {
			return "", nil
		}
		html, _ := articleSel.Html()
		return html, nil
	})
	if err != nil {
		return "", err
	}

	contentHTML, ok := cachedValue.(string)
	if !ok || contentHTML == "" {
		return "", nil
	}

	cleaned, err := routeutils.CleanDescription(contentHTML, rootURL, routeutils.DefaultCleanOptions())
	if err != nil {
		return contentHTML, nil
	}

	return extractBodyHTML(cleaned), nil
}

func extractBodyHTML(cleaned string) string {
	doc, err := parser.LoadString(cleaned)
	if err != nil {
		return cleaned
	}

	bodySel := doc.Find("body")
	if bodySel.Length() == 0 {
		return cleaned
	}

	bodyHTML, _ := bodySel.Html()
	if bodyHTML == "" {
		return cleaned
	}
	return bodyHTML
}
