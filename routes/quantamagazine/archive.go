package routes

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

const quantaRootURL = "https://www.quantamagazine.org"

// quantaProfile disguises article page fetches; the WP REST API tolerates any client.
var quantaProfile = disguise.Chrome()

var quantaLatexRe = regexp.MustCompile(`\$latex([\S\s]+?)\$`)

type quantaPost struct {
	ID    int    `json:"id"`
	Date  string `json:"date"`
	Link  string `json:"link"`
	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Embedded *struct {
		Author []struct {
			Name string `json:"name"`
			Code string `json:"code"` // present when the embed lookup fails
		} `json:"author"`
	} `json:"_embedded,omitempty"`
}

var archiveRoute = routeutils.RouteSpec{
	Path:        "archive",
	Name:        "Archive",
	Example:     "quantamagazine/archive",
	Maintainers: []string{"xihale"},
	URL:         "https://www.quantamagazine.org",
	Description: "Latest articles from Quanta Magazine, with full article content",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "default 10, max 20"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  ArchiveHandler,
}

// ArchiveHandler handles /quantamagazine/archive
func ArchiveHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 10, 20)

	apiURL := fmt.Sprintf("%s/wp-json/wp/v2/posts?per_page=%d&page=1&_embed=author", quantaRootURL, limit)
	var posts []quantaPost
	if err := routeutils.GetJSON(c.Parent(), c.Client(), apiURL, &posts); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("Quanta Magazine", quantaRootURL+"/", "Illuminating basic science and math research")

	// Fetch full article bodies with bounded concurrency.
	bodies := make([]string, len(posts))
	authors := make([]string, len(posts))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for i := range posts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			body, author := fetchQuantaArticle(c, posts[i].Link)
			bodies[i] = body
			authors[i] = author
		}(i)
	}
	wg.Wait()

	for i, post := range posts {
		title := html.UnescapeString(post.Title.Rendered)
		if title == "" || post.Link == "" {
			continue
		}
		pubDate, _ := dateutil.ParseDate(post.Date)
		if pubDate.IsZero() && post.Date != "" {
			// WP REST API emits "2006-01-02T15:04:05" without a zone.
			if t, err := time.ParseInLocation("2006-01-02T15:04:05", post.Date, time.UTC); err == nil {
				pubDate = t
			}
		}
		item := routeutils.NewItem(title, post.Link, bodies[i], pubDate)
		item.GUID = post.Link
		authorName := authors[i]
		if authorName == "" && post.Embedded != nil {
			for _, a := range post.Embedded.Author {
				if a.Name != "" {
					authorName = a.Name
					break
				}
			}
		}
		if authorName != "" {
			routeutils.SetItemAuthor(item, authorName, "", "")
		}
		routeutils.AddItem(feed, item)
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("quantamagazine: no posts returned by WP API")
	}
	return feed, nil
}

// fetchQuantaArticle downloads an article page and extracts the #postBody HTML.
// It returns the processed HTML and the byline author name (either may be empty).
func fetchQuantaArticle(c *ctxpkg.Context, link string) (string, string) {
	doc, err := quantaProfile.Fetch(link).GetHTML(c.Parent(), c.Client())
	if err != nil || doc == nil {
		return "", ""
	}

	author := ""
	if byline := doc.FindSelector("span.byline__author"); byline != nil {
		author = strings.TrimSpace(byline.Text())
	}

	body := doc.FindSelector("#postBody")
	if body == nil {
		return "", author
	}

	clone := body.Clone()
	for _, sel := range []string{
		".header-spacer", ".scale1.mha", ".post__title__author-date",
		".post__aside--divider", ".hide-on-print", ".post__aside__pullquote",
		"aside.post__sidebar.hide", "nav[data-glide-el]", ".post__footer",
		".iframe-placeholder",
	} {
		if junk := clone.Find(sel); junk != nil {
			junk.Remove()
		}
	}

	content, err := clone.Html()
	if err != nil {
		return "", author
	}
	content = quantaLatexRe.ReplaceAllString(content, `<img align="center" src="https://latex.codecogs.com/png.latex?$1"/>`)
	content = strings.TrimSpace(content)
	return content, author
}
