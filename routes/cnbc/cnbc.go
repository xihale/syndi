package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const (
	cnbcFeedBase   = "https://search.cnbc.com/rs/search/combinedcms/view.xml"
	cnbcDefaultID  = "100003114" // Top News
	cnbcDefaultTTL = 15 * time.Minute
)

var cnbcRSSRoute = routeutils.RouteSpec{
	Path:        "rss/:id",
	Name:        "CNBC Full Article RSS",
	Example:     "cnbc/rss/-",
	Maintainers: []string{"xihale"},
	Description: "CNBC channel feed with full article text. Channel IDs come from the official RSS URLs at https://www.cnbc.com/rss-feeds/; '-' selects Top News (100003114)",
	Categories:  []models.Category{{Name: "finance"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Numeric channel ID from the official RSS URL, e.g. 100003114 (Top News), 10001147 (Business), 10000664 (Markets); use '-' for Top News"),
	},
	CacheTTL: cnbcDefaultTTL,
	Handler:  CNBCRSSHandler,
}

// CNBCRSSHandler handles /cnbc/rss/:id
func CNBCRSSHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "-" || id == "" {
		id = cnbcDefaultID
	}
	feedURL := fmt.Sprintf("%s?partnerId=wrss01&id=%s", cnbcFeedBase, id)

	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), feedURL)
	if err != nil {
		return nil, err
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items for CNBC channel %s", id)
	}

	ctx := c.Parent()
	for i := range feed.Items {
		item := &feed.Items[i]
		if item.Link == "" || strings.HasPrefix(item.Link, "https://www.cnbc.com/select/") {
			continue
		}
		enrichCNBCItem(ctx, c.Client(), item)
	}
	return feed, nil
}

// enrichCNBCItem fetches the article page and fills in body text, author and keywords.
func enrichCNBCItem(ctx context.Context, cl *client.Client, item *models.Item) {
	doc, err := routeutils.GetHTML(ctx, cl, item.Link)
	if err != nil || doc == nil {
		return // keep teaser description from the feed
	}

	var parts []string
	if kp := doc.Find(".RenderKeyPoints-keyPoints").First(); kp.Length() > 0 {
		if h, kerr := kp.Html(); kerr == nil && strings.TrimSpace(h) != "" {
			parts = append(parts, h)
		}
	}
	for _, sel := range []string{".FeaturedContent-articleBody", ".ArticleBody-articleBody", ".LiveBlogBody-articleBody"} {
		if b := doc.Find(sel).First(); b.Length() > 0 {
			if h, berr := b.Html(); berr == nil && strings.TrimSpace(h) != "" {
				parts = append(parts, h)
				break
			}
		}
	}
	if len(parts) > 0 {
		item.Description = strings.Join(parts, "<br/>")
	}

	// Author and keywords from the last JSON-LD NewsArticle block.
	author, keywords := cnbcMetaFromJSONLD(doc)
	if author != "" {
		routeutils.SetAuthor(item, author)
	}
	if len(keywords) > 0 {
		routeutils.SetCategories(item, keywords...)
	}
}

func cnbcMetaFromJSONLD(doc *parser.Document) (string, []string) {
	var author string
	var keywords []string
	var scripts []string
	doc.Find("script[type='application/ld+json']").Each(func(_ int, s *goquery.Selection) {
		scripts = append(scripts, s.Text())
	})
	for _, raw := range scripts {
		var meta map[string]any
		if err := json.Unmarshal([]byte(raw), &meta); err != nil {
			continue
		}
		if t, _ := meta["@type"].(string); t != "NewsArticle" {
			continue
		}
		author = cnbcAuthorNames(meta["author"])
		keywords = cnbcKeywordList(meta["keywords"])
	}
	return author, keywords
}

func cnbcAuthorNames(v any) string {
	switch a := v.(type) {
	case map[string]any:
		if n, _ := a["name"].(string); n != "" {
			return html.UnescapeString(n)
		}
	case []any:
		var names []string
		for _, e := range a {
			if m, ok := e.(map[string]any); ok {
				if n, _ := m["name"].(string); n != "" {
					names = append(names, n)
				}
			}
		}
		return strings.Join(names, ", ")
	}
	return ""
}

func cnbcKeywordList(v any) []string {
	switch k := v.(type) {
	case string:
		var out []string
		for _, part := range strings.Split(k, ",") {
			if p := strings.TrimSpace(html.UnescapeString(part)); p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		var out []string
		for _, e := range k {
			if s, ok := e.(string); ok {
				if p := strings.TrimSpace(html.UnescapeString(s)); p != "" {
					out = append(out, p)
				}
			}
		}
		return out
	}
	return nil
}
