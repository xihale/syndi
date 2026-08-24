package routes

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/disguise"
	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

// reddit now 403s non-browser JSON requests from many networks; the Firefox
// profile with a browser Accept header gets the listing through.
var redditProfile = disguise.Firefox().
	WithHeader("Accept", "application/json, text/plain, */*")

var redditSubredditRoute = routeutils.RouteSpec{
	Path:        ":subreddit",
	Name:        "Reddit Subreddit",
	Example:     "reddit/golang",
	Maintainers: []string{"xihale"},
	Description: "Fetch posts from a Reddit subreddit",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("subreddit", "Subreddit name (without r/)"),
	},
	CacheTTL: 30 * time.Minute, // Reddit rate-limits hard; cache aggressively
	Handler:  RedditSubredditHandler,
}

// RedditSubredditHandler handles /reddit/:subreddit.
// Reddit 403/429s unauthenticated .json listing APIs from many networks, so we
// consume the more permissive native .rss feed through a browser disguise
// profile and normalize it into our Feed shape.
func RedditSubredditHandler(c *ctxpkg.Context) (*models.Feed, error) {
	subreddit := c.Param("subreddit")
	sortBy := parseRedditSort(c.QueryParam("sort"))
	limit := parseRedditLimit(c.QueryParam("limit"))
	timeRange := parseRedditTimeRange(c.QueryParam("t"))

	feedURL := buildRedditListingURL(subreddit, sortBy, limit, timeRange)

	feed, err := redditProfile.Fetch(feedURL).GetFeed(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}

	feed.Title = fmt.Sprintf("Reddit r/%s", subreddit)
	feed.Link = fmt.Sprintf("https://www.reddit.com/r/%s", subreddit)
	feed.Description = fmt.Sprintf("%s posts from r/%s", sortDisplayName(sortBy), subreddit)

	trimmed := make([]models.Item, 0, min(limit, len(feed.Items)))
	for _, item := range feed.Items {
		item.Categories = append(item.Categories, "r/"+subreddit)
		trimmed = append(trimmed, item)
		if len(trimmed) >= limit {
			break
		}
	}
	feed.Items = trimmed
	return feed, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseRedditSort(raw string) string {
	return routeutils.ParseEnum(raw, "hot", "hot", "new", "top", "rising")
}

func parseRedditLimit(raw string) int {
	return routeutils.ParsePositiveInt(raw, 25, 100)
}

func parseRedditTimeRange(raw string) string {
	return routeutils.ParseEnum(raw, "day", "hour", "day", "week", "month", "year", "all")
}

func buildRedditListingURL(subreddit, sortBy string, limit int, timeRange string) string {
	sub := url.PathEscape(subreddit)
	switch sortBy {
	case "hot":
		return fmt.Sprintf("https://www.reddit.com/r/%s/.rss", sub)
	case "new":
		return fmt.Sprintf("https://www.reddit.com/r/%s/new/.rss", sub)
	case "rising":
		return fmt.Sprintf("https://www.reddit.com/r/%s/rising/.rss", sub)
	default: // top
		return fmt.Sprintf("https://www.reddit.com/r/%s/top/.rss?t=%s", sub, url.QueryEscape(timeRange))
	}
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
