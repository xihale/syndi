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
	cacheTTL := 10 * time.Minute // Reddit is very active, shorter cache

	route := &models.Route{
		Path:         "/reddit/:subreddit",
		Name:         "Reddit Subreddit",
		Example:      "reddit/golang",
		Maintainers:  []string{"yourname"},
		Description:  "Fetch posts from a Reddit subreddit",
		Categories:   []models.Category{{Name: "social-media"}},
		Features:     models.Features{SupportRadar: true},
		Handler:      RedditSubredditHandler,
		Parameters: []models.Parameter{
			{Name: "subreddit", Required: true, Description: "Subreddit name (without r/)"},
		},
		CacheTTL: &cacheTTL,
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// RedditSubredditHandler handles /reddit/:subreddit
func RedditSubredditHandler(c *ctxpkg.Context) (*models.Feed, error) {
	subreddit := c.Param("subreddit")
	ctx := c.Parent()

	url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=25", subreddit)

	var response RedditResponse
	if err := routeutils.GetJSON(ctx, c.Client(), url, &response); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Reddit r/%s", subreddit),
		fmt.Sprintf("https://www.reddit.com/r/%s", subreddit),
		fmt.Sprintf("Hot posts from r/%s", subreddit),
	)

	for _, post := range response.Data.Children {
		if post.Kind == "t3" {
			// Skip stickied posts
			if post.Data.Stickied {
				continue
			}

			// Use external URL or permalink
			link := post.Data.URL
			if link == "" {
				link = "https://www.reddit.com" + post.Data.Permalink
			}

			item := routeutils.NewItem(
				post.Data.Title,
				link,
				post.Data.SelftextHTML,
				time.Unix(int64(post.Data.CreatedUTC), 0),
			)
			item.GUID = "reddit-" + post.Data.ID

			// Set author
			if post.Data.Author != "" {
				routeutils.SetAuthor(item, post.Data.Author,
					routeutils.WithAuthorURI("https://www.reddit.com/user/"+post.Data.Author))
			}

			// Add categories (subreddit and flair)
			routeutils.SetCategories(item, "r/"+subreddit)
			if post.Data.LinkFlairText != "" {
				routeutils.SetCategories(item, post.Data.LinkFlairText)
			}

			routeutils.AddItem(feed, item)
		}
	}

	return feed, nil
}

type RedditResponse struct {
	Data struct {
		Children []RedditChild `json:"children"`
	} `json:"data"`
}

type RedditChild struct {
	Kind string       `json:"kind"`
	Data RedditPost   `json:"data"`
}

type RedditPost struct {
	Kind          string  `json:"kind"`
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Selftext      string  `json:"selftext"`
	SelftextHTML  string  `json:"selftext_html"`
	URL           string  `json:"url"`
	Permalink     string  `json:"permalink"`
	Author        string  `json:"author"`
	CreatedUTC    float64 `json:"created_utc"`
	Stickied      bool    `json:"stickied"`
	LinkFlairText string  `json:"link_flair_text"`
	Over18        bool    `json:"over_18"`
}
