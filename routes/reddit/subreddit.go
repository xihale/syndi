package routes

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
)

func init() {
	cacheTTL := 10 * time.Minute // Reddit is very active, shorter cache

	route := &models.Route{
		Path:        "/reddit/:subreddit",
		Name:        "Reddit Subreddit",
		Example:     "reddit/golang",
		Maintainers: []string{"yourname"},
		Description: "Fetch posts from a Reddit subreddit",
		Categories:  []models.Category{{Name: "social-media"}},
		Features:    models.Features{SupportRadar: true},
		Handler:     RedditSubredditHandler,
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
	sortBy := parseRedditSort(c.QueryParam("sort"))
	limit := parseRedditLimit(c.QueryParam("limit"))
	timeRange := parseRedditTimeRange(c.QueryParam("t"))

	ctx, cancel := context.WithTimeout(c.Parent(), 10*time.Second)
	defer cancel()

	url := buildRedditListingURL(subreddit, sortBy, limit, timeRange)

	var response RedditResponse
	if err := routeutils.GetJSON(ctx, c.Client(), url, &response); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Reddit r/%s", subreddit),
		fmt.Sprintf("https://www.reddit.com/r/%s", subreddit),
		fmt.Sprintf("%s posts from r/%s", sortDisplayName(sortBy), subreddit),
	)
	feed.Items = make([]models.Item, 0, limit)

	for _, post := range response.Data.Children {
		if post.Kind != "t3" || post.Data.Stickied {
			continue
		}
		if len(feed.Items) >= limit {
			break
		}

		link := resolveRedditPostLink(post.Data.URL, post.Data.Permalink)
		item := routeutils.NewItem(
			post.Data.Title,
			link,
			buildRedditDescription(post.Data),
			time.Unix(int64(post.Data.CreatedUTC), 0),
		)
		item.GUID = "reddit-" + post.Data.ID

		if post.Data.Author != "" {
			routeutils.SetAuthor(item, post.Data.Author,
				routeutils.WithAuthorURI("https://www.reddit.com/user/"+post.Data.Author))
		}

		routeutils.SetCategories(item, "r/"+subreddit)
		if post.Data.LinkFlairText != "" {
			routeutils.SetCategories(item, post.Data.LinkFlairText)
		}
		if post.Data.Over18 {
			routeutils.SetCategories(item, "NSFW")
		}

		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

func parseRedditSort(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "new", "top", "rising":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "hot"
	}
}

func parseRedditLimit(raw string) int {
	return clampPositive(raw, 25, 100)
}

func parseRedditTimeRange(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "hour", "day", "week", "month", "year", "all":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "day"
	}
}

func buildRedditListingURL(subreddit, sortBy string, limit int, timeRange string) string {
	endpoint := fmt.Sprintf(
		"https://www.reddit.com/r/%s/%s.json?limit=%d",
		url.PathEscape(subreddit),
		sortBy,
		limit,
	)
	if sortBy == "top" {
		return endpoint + "&t=" + url.QueryEscape(timeRange)
	}
	return endpoint
}

func resolveRedditPostLink(postURL, permalink string) string {
	postURL = strings.TrimSpace(postURL)
	if postURL == "" {
		return "https://www.reddit.com" + permalink
	}
	if strings.HasPrefix(postURL, "/") {
		return "https://www.reddit.com" + postURL
	}
	return postURL
}

func buildRedditDescription(post RedditPost) string {
	if post.SelftextHTML != "" {
		return routeutils.DecodeHTMLEntities(post.SelftextHTML)
	}
	return post.Selftext
}

func sortDisplayName(sortBy string) string {
	if sortBy == "" {
		return "Hot"
	}
	return strings.ToUpper(sortBy[:1]) + sortBy[1:]
}

func clampPositive(raw string, defaultValue, maxValue int) int {
	parsed := defaultValue
	if raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			parsed = v
		}
	}
	if parsed > maxValue {
		return maxValue
	}
	return parsed
}

type RedditResponse struct {
	Data struct {
		Children []RedditChild `json:"children"`
	} `json:"data"`
}

type RedditChild struct {
	Kind string     `json:"kind"`
	Data RedditPost `json:"data"`
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
