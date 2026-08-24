// Package routes implements RSSHub-style routes for yande.re (moebooru).
package routes

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var yanderePostRoute = routeutils.RouteSpec{
	Path:        "post",
	Name:        "yande.re Posts",
	Example:     "yandere/post",
	Maintainers: []string{"xihale"},
	Description: "Latest yande.re posts, safe-rated by default",
	Categories:  []models.Category{{Name: "picture"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("tags", `Extra search tags; a safe-rating filter is added unless the tags contain their own rating:`),
		routeutils.OptionalParam("rating", `Rating filter overriding the default ("safe", "questionable", "explicit")`),
		routeutils.OptionalParam("limit", "Number of posts (default 20, max 100)"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  YanderePostHandler,
}

type moebooruPost struct {
	ID         int64  `json:"id"`
	CreatedAt  int64  `json:"created_at"` // unix epoch seconds
	Tags       string `json:"tags"`
	Rating     string `json:"rating"`
	Score      int    `json:"score"`
	Author     string `json:"author"`
	Source     string `json:"source"`
	FileURL    string `json:"file_url"`
	SampleURL  string `json:"sample_url"`
	PreviewURL string `json:"preview_url"`
	MD5        string `json:"md5"`
}

// YanderePostHandler handles /yandere/post
func YanderePostHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 100)
	tags := strings.TrimSpace(c.QueryParam("tags"))
	rating := strings.TrimSpace(c.QueryParam("rating"))

	searchTags := buildMoebooruTags(tags, rating, "rating:safe")

	ctx, cancel := context.WithTimeout(c.Parent(), 30*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf("https://yande.re/post.json?limit=%d&tags=%s", limit, url.QueryEscape(searchTags))

	var posts []moebooruPost
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &posts); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"yande.re",
		"https://yande.re/post",
		"Latest posts from yande.re",
	)
	routeutils.AppendMappedItems(feed, posts, limit, func(p moebooruPost) *models.Item {
		return newMoebooruItem(p, "yande.re", "yandere", "https://yande.re/post/show/%d")
	})

	return feed, nil
}

func buildMoebooruTags(userTags, rating, defaultRating string) string {
	var parts []string
	if userTags != "" {
		parts = append(parts, userTags)
	}
	switch {
	case rating != "":
		parts = append(parts, "rating:"+rating)
	case !strings.Contains(userTags, "rating:"):
		parts = append(parts, defaultRating)
	}
	return strings.Join(parts, ",")
}

func newMoebooruItem(p moebooruPost, feedName, guidPrefix, linkFmt string) *models.Item {
	if p.ID == 0 || p.PreviewURL == "" {
		return nil
	}
	link := fmt.Sprintf(linkFmt, p.ID)

	image := p.SampleURL
	if image == "" {
		image = p.FileURL
	}
	tags := firstNFields(p.Tags, 5)

	var b strings.Builder
	fmt.Fprintf(&b, `<img src="%s"/>`, html.EscapeString(image))
	if len(tags) > 0 {
		b.WriteString("<p>Tags: " + html.EscapeString(strings.Join(tags, ", ")) + "</p>")
	}
	b.WriteString("<p>Score: " + html.EscapeString(strconv.Itoa(p.Score)) + "</p>")

	title := fmt.Sprintf("Post #%d", p.ID)
	if len(tags) > 0 {
		title = fmt.Sprintf("Post #%d - %s", p.ID, strings.Join(tags[:min(2, len(tags))], ", "))
	}
	item := routeutils.NewItem(title, link, b.String(), time.Unix(p.CreatedAt, 0))
	if p.MD5 != "" {
		item.GUID = guidPrefix + "-" + p.MD5
	} else {
		item.GUID = fmt.Sprintf("%s-%d", guidPrefix, p.ID)
	}
	if p.Author != "" {
		routeutils.SetItemAuthor(item, p.Author, "", "")
	}
	routeutils.SetCategories(item, tags...)
	return item
}

func firstNFields(s string, n int) []string {
	fields := strings.Fields(s)
	if len(fields) > n {
		return fields[:n]
	}
	return fields
}
