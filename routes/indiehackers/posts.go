package routes

import (
	"fmt"
	"time"

	ctxpkg "github.com/rsshub/go/pkg/context"
	"github.com/rsshub/go/pkg/models"
	"github.com/rsshub/go/pkg/registry"
	"github.com/rsshub/go/internal/routeutils"
)

func init() {
	route := &models.Route{
		Path:         "/indiehackers/:type",
		Name:         "Indie Hackers",
		Example:      "indiehackers/posts",
		Maintainers:  []string{"yourname"},
		Description:  "Fetch content from Indie Hackers community",
		Categories:   []models.Category{{Name: "programming"}, {Name: "social-media"}},
		Features:     models.Features{SupportRadar: true},
		Handler:      IndieHackersHandler,
		Parameters: []models.Parameter{
			{Name: "type", Required: false, Description: "Content type: posts, questions, products (default: posts)"},
		},
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// IndieHackersHandler handles /indiehackers/:type
func IndieHackersHandler(c *ctxpkg.Context) (*models.Feed, error) {
	contentType := c.Param("type")
	if contentType == "" {
		contentType = "posts"
	}

	ctx := c.Parent()

	// Note: This is a hypothetical API - actual Indie Hackers API may differ
	url := fmt.Sprintf("https://www.indiehackers.com/api/%s?sortBy=recent&limit=20", contentType)

	var items []IndieHackersItem
	if err := routeutils.GetJSON(ctx, c.Client(), url, &items); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Indie Hackers %s", contentType),
		fmt.Sprintf("https://www.indiehackers.com/%s", contentType),
		fmt.Sprintf("Latest %s from Indie Hackers", contentType),
	)

	for _, item := range items {
		description := item.Summary
		if item.Body != "" {
			description = item.Body
		}

		pubDate := item.CreatedAt
		if pubDate.IsZero() {
			pubDate = time.Now()
		}

		feedItem := routeutils.NewItem(
			item.Title,
			item.URL,
			description,
			pubDate,
		)
		feedItem.GUID = fmt.Sprintf("indiehackers-%s", item.ID)

		// Set author
		if item.Author.Name != "" {
			routeutils.SetAuthor(feedItem, item.Author.Name,
				routeutils.WithAuthorURI(item.Author.URL))
		}

		// Add categories (tags)
		if len(item.Tags) > 0 {
			categories := make([]string, len(item.Tags))
			for i, tag := range item.Tags {
				categories[i] = tag.Name
			}
			routeutils.SetCategories(feedItem, categories...)
		}

		// Add engagement as categories
		if item.PointsCount > 0 {
			routeutils.SetCategories(feedItem, fmt.Sprintf("%d points", item.PointsCount))
		}
		if item.CommentsCount > 0 {
			routeutils.SetCategories(feedItem, fmt.Sprintf("%d comments", item.CommentsCount))
		}

		routeutils.AddItem(feed, feedItem)
	}

	return feed, nil
}

type IndieHackersItem struct {
	ID           string             `json:"id"`
	Title        string             `json:"title"`
	URL          string             `json:"url"`
	Summary      string             `json:"summary"`
	Body         string             `json:"body"`
	CreatedAt    time.Time          `json:"created_at"`
	PointsCount  int                `json:"points_count"`
	CommentsCount int               `json:"comments_count"`
	Author       IndieHackersAuthor `json:"author"`
	Tags         []IndieHackersTag  `json:"tags"`
}

type IndieHackersAuthor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type IndieHackersTag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
