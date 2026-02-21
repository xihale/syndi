package routes

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/parser"
	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
	"github.com/xihale/rsshub-go/pkg/utils/date"
)

func init() {
	cacheTTL := 5 * 24 * time.Hour // Blog list cache: 5 days

	route := &models.Route{
		Path:        "/techne98/blog",
		Name:        "techne98 - blog",
		Example:     "techne98/blog",
		Maintainers: []string{"xihale"},
		Description: "Fetch blog posts from techne98.com",
		Categories:  []models.Category{{Name: "blog"}},
		Features:    models.Features{},
		Handler:     Techne98BlogHandler,
		Parameters: []models.Parameter{
			{Name: "limit", Required: false, Description: "Maximum number of articles to return (default: all)"},
		},
		CacheTTL: &cacheTTL,
	}

	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
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

	var items []*models.Item
	var maxItems *int

	limitStr := c.QueryParam("limit")
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			maxItems = &parsed
		}
	}

	postCount := 0
	doc.Each("main .grid a.group", func(i int, sel *parser.Selection) {
		if maxItems != nil && postCount >= *maxItems {
			sel.Selection.End()
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

		contentCacheKey := fmt.Sprintf("techne98:article:%s", link)
		contentHTMLVal, err := c.CacheTryGet(contentCacheKey, 30*24*time.Hour, func() (interface{}, error) {
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
		if err == nil && contentHTMLVal != "" {
			contentHTML := contentHTMLVal.(string)
			if contentHTML != "" {
				cleaned, err := routeutils.CleanDescription(contentHTML, rootURL, routeutils.DefaultCleanOptions())
				if err == nil {
					doc, parseErr := parser.LoadString(cleaned)
					if parseErr == nil {
						bodySel := doc.Find("body")
						if bodySel.Length() > 0 {
							bodyHTML, _ := bodySel.Html()
							if bodyHTML != "" {
								item.Description = bodyHTML
							} else {
								item.Description = cleaned
							}
						} else {
							item.Description = cleaned
						}
					} else {
						item.Description = cleaned
					}
				} else {
					item.Description = contentHTML
				}
			}
		}

		items = append(items, item)
		postCount++
	})

	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	for _, item := range items {
		routeutils.AddItem(feed, item)
	}

	return feed, nil
}
