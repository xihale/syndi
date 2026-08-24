// Package routes implements RSSHub-style routes for Danbooru.
package routes

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var danbooruPostsRoute = routeutils.RouteSpec{
	Path:        "posts",
	Name:        "Danbooru Posts",
	Example:     "danbooru/posts",
	Maintainers: []string{"xihale"},
	Description: "Latest Danbooru posts, safe-rated by default",
	Categories:  []models.Category{{Name: "picture"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("tags", `Extra search tags (e.g. "touhou"); a safe-rating filter is added unless the tags contain their own rating:`),
		routeutils.OptionalParam("rating", `Rating filter overriding the default (e.g. "g", "general", "s")`),
		routeutils.OptionalParam("limit", "Number of posts (default 20, max 100)"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  DanbooruPostsHandler,
}

type danbooruPost struct {
	ID               int64     `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	Rating           string    `json:"rating"`
	Source           string    `json:"source"`
	PreviewFileURL   string    `json:"preview_file_url"`
	LargeFileURL     string    `json:"large_file_url"`
	FileURL          string    `json:"file_url"`
	TagStringGeneral string    `json:"tag_string_general"`
	TagStringArtist  string    `json:"tag_string_artist"`
}

// DanbooruPostsHandler handles /danbooru/posts
func DanbooruPostsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 100)
	tags := strings.TrimSpace(c.QueryParam("tags"))
	rating := strings.TrimSpace(c.QueryParam("rating"))

	searchTags := buildDanbooruTags(tags, rating)

	ctx, cancel := context.WithTimeout(c.Parent(), 30*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf("https://danbooru.donmai.us/posts.json?limit=%d&tags=%s", limit, url.QueryEscape(searchTags))

	var posts []danbooruPost
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &posts); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"Danbooru",
		"https://danbooru.donmai.us/posts",
		"Latest posts from Danbooru",
	)
	routeutils.AppendMappedItems(feed, posts, limit, func(p danbooruPost) *models.Item {
		return newDanbooruItem(p)
	})

	return feed, nil
}

func buildDanbooruTags(userTags, rating string) string {
	var parts []string
	if userTags != "" {
		parts = append(parts, userTags)
	}
	switch {
	case rating != "":
		parts = append(parts, "rating:"+rating)
	case !strings.Contains(userTags, "rating:"):
		parts = append(parts, "rating:g,general")
	}
	return strings.Join(parts, ",")
}

func newDanbooruItem(p danbooruPost) *models.Item {
	if p.ID == 0 || p.PreviewFileURL == "" {
		return nil
	}
	link := fmt.Sprintf("https://danbooru.donmai.us/posts/%d", p.ID)

	image := p.PreviewFileURL
	if image == "" && p.LargeFileURL != "" {
		image = p.LargeFileURL
	}
	tags := firstNFields(p.TagStringGeneral, 5)

	var b strings.Builder
	fmt.Fprintf(&b, `<img src="%s"/>`, html.EscapeString(image))
	if len(tags) > 0 {
		b.WriteString("<p>Tags: " + html.EscapeString(strings.Join(tags, ", ")) + "</p>")
	}
	if p.Source != "" {
		b.WriteString(`<p>Source: <a href="` + html.EscapeString(p.Source) + `">` + html.EscapeString(p.Source) + "</a></p>")
	}

	title := fmt.Sprintf("Post #%d", p.ID)
	if len(tags) > 0 {
		title = fmt.Sprintf("Post #%d - %s", p.ID, strings.Join(tags[:min(2, len(tags))], ", "))
	}
	item := routeutils.NewItem(title, link, b.String(), p.CreatedAt)
	item.GUID = fmt.Sprintf("danbooru-%d", p.ID)
	if artist := firstNFields(p.TagStringArtist, 1); len(artist) == 1 {
		routeutils.SetItemAuthor(item, artist[0], "", "")
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
