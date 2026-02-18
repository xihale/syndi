package routes

import (
	"fmt"
	"strconv"
	"time"

	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
	"github.com/xihale/rsshub-go/internal/routeutils"
)

func init() {
	cacheTTL := 1 * time.Hour // Hacker News updates infrequently

	route := &models.Route{
		Path:         "/hackernews/stories",
		Name:         "Hacker News Top Stories",
		Example:      "hackernews/stories",
		Maintainers:  []string{"yourname"},
		Description:  "Fetch top stories from Hacker News",
		Categories:   []models.Category{{Name: "social-media"}, {Name: "it"}},
		Features:     models.Features{},
		Handler:      HackerNewsStoriesHandler,
		CacheTTL:     &cacheTTL,
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// HackerNewsStoriesHandler handles /hackernews/stories
func HackerNewsStoriesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()

	// Get top story IDs
	var storyIDs []int
	if err := routeutils.GetJSON(ctx, c.Client(), "https://hacker-news.firebaseio.com/v0/topstories.json", &storyIDs); err != nil {
		return nil, err
	}

	// Limit to first 30 stories
	if len(storyIDs) > 30 {
		storyIDs = storyIDs[:30]
	}

	feed := routeutils.NewFeed(
		"Hacker News Top Stories",
		"https://news.ycombinator.com/news",
		"Top stories from Hacker News",
	)

	// Fetch each story
	for _, id := range storyIDs {
		var story HNStory
		url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)

		if err := routeutils.GetJSON(ctx, c.Client(), url, &story); err != nil {
			continue
		}

		if story.URL == "" {
			story.URL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)
		}

		item := routeutils.NewItem(
			story.Title,
			story.URL,
			story.Text,
			time.Unix(story.Time, 0),
		)
		item.GUID = fmt.Sprintf("hn-%d", story.ID)

		// Set author if available
		if story.By != "" {
			routeutils.SetAuthor(item, story.By, routeutils.WithAuthorURI("https://news.ycombinator.com/user?id="+story.By))
		}

		// Add score as category
		routeutils.SetCategories(item, "Score: "+strconv.Itoa(story.Score))

		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

type HNStory struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Score int    `json:"score"`
	By    string `json:"by"`
	Time  int64  `json:"time"`
	Text  string `json:"text"`
}
