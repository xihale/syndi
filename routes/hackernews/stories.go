package routes

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var hackerNewsStoriesRoute = routeutils.RouteSpec{
	Path:        "/hackernews/stories",
	Name:        "Hacker News Top Stories",
	Example:     "hackernews/stories",
	Maintainers: []string{"xihale"},
	Description: "Fetch top stories from Hacker News",
	Categories:  []models.Category{{Name: "social-media"}, {Name: "it"}},
	Features:    models.Features{},
	CacheTTL:    1 * time.Hour, // Hacker News updates infrequently
	Handler:     HackerNewsStoriesHandler,
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

	type storyJob struct {
		index int
		id    int
	}
	jobs := make(chan storyJob)
	items := make([]*models.Item, len(storyIDs))

	workerCount := 8
	if len(storyIDs) < workerCount {
		workerCount = len(storyIDs)
	}

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				item, err := fetchHNStoryItem(ctx, c, job.id)
				if err != nil || item == nil {
					continue
				}
				items[job.index] = item
			}
		}()
	}

enqueue:
	for i, id := range storyIDs {
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- storyJob{index: i, id: id}:
		}
	}
	close(jobs)
	wg.Wait()

	for _, item := range items {
		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

func fetchHNStoryItem(ctx context.Context, c *ctxpkg.Context, id int) (*models.Item, error) {
	var story HNStory
	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)

	if err := routeutils.GetJSON(ctx, c.Client(), url, &story); err != nil {
		return nil, err
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

	if story.By != "" {
		routeutils.SetAuthor(item, story.By, routeutils.WithAuthorURI("https://news.ycombinator.com/user?id="+story.By))
	}

	routeutils.SetCategories(item, "Score: "+strconv.Itoa(story.Score))
	return item, nil
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
