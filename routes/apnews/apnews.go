package routes

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

const (
	apNewsSitemapLatest  = "https://apnews.com/news-sitemap-latest.xml"
	apNewsSitemapContent = "https://apnews.com/news-sitemap-content.xml"
	apNewsHomePage       = "https://apnews.com"
)

// apSitemapEntry is one <url> of AP's Google News sitemap.
type apSitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
	News    struct {
		Title           string `xml:"http://www.google.com/schemas/sitemap-news/0.9 title"`
		PublicationDate string `xml:"http://www.google.com/schemas/sitemap-news/0.9 publication_date"`
		Publication     struct {
			Name     string `xml:"http://www.google.com/schemas/sitemap-news/0.9 name"`
			Language string `xml:"http://www.google.com/schemas/sitemap-news/0.9 language"`
		} `xml:"http://www.google.com/schemas/sitemap-news/0.9 publication"`
	} `xml:"http://www.google.com/schemas/sitemap-news/0.9 news"`
}

type apSitemapDoc struct {
	XMLName xml.Name         `xml:"urlset"`
	URLs    []apSitemapEntry `xml:"url"`
}

var apNewsRoute = routeutils.RouteSpec{
	Path:        "latest",
	Name:        "AP News Latest",
	Example:     "apnews/latest",
	Maintainers: []string{"xihale"},
	Description: "AP News latest articles from the official Google News sitemaps. Optional query params: limit (default 30), lang (eng|spa, no filter by default), fulltext=true to fetch article bodies",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{},
	Parameters:  nil,
	CacheTTL:    15 * time.Minute,
	Handler:     APNewsHandler,
}

// APNewsHandler handles /apnews/latest
func APNewsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)
	fulltext := routeutils.ParseBool(c.QueryParam("fulltext"), false)
	lang := strings.TrimSpace(c.QueryParam("lang"))

	entries := make([]apSitemapEntry, 0, 512)
	for _, sm := range []string{apNewsSitemapLatest, apNewsSitemapContent} {
		data, err := routeutils.FetchBytes(ctx, c.Client(), sm)
		if err != nil {
			return nil, err
		}
		var doc apSitemapDoc
		if err := routeutils.UnmarshalXML(data, &doc); err != nil {
			return nil, fmt.Errorf("parse AP sitemap %s: %w", sm, err)
		}
		entries = append(entries, doc.URLs...)
	}

	feed := routeutils.NewFeed("AP News", apNewsHomePage+"/", "Associated Press latest news")

	type datedItem struct {
		item *models.Item
		date time.Time
	}
	items := make([]datedItem, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		link := strings.TrimSpace(e.Loc)
		title := strings.TrimSpace(e.News.Title)
		if link == "" || seen[link] {
			continue
		}
		if title == "" || strings.Contains(link, "/hub/") {
			continue
		}
		seen[link] = true
		if lang != "" && e.News.Publication.Language != "" && e.News.Publication.Language != lang {
			continue
		}
		var pub time.Time
		raw := firstNonEmptyString(e.News.PublicationDate, e.LastMod)
		if raw != "" {
			if t, err := dateutil.ParseDate(strings.TrimSpace(raw)); err == nil {
				pub = t
			}
		}
		item := routeutils.NewItem(title, link, "", pub)
		item.GUID = link
		if l := e.News.Publication.Language; l != "" {
			routeutils.SetCategories(item, l)
		}
		items = append(items, datedItem{item: item, date: pub})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].date.After(items[j].date) })

	count := 0
	for _, di := range items {
		if count >= limit {
			break
		}
		if fulltext {
			apEnrichWithFullText(ctx, c.Client(), di.item)
		}
		routeutils.AddItem(feed, di.item)
		count++
	}
	return feed, nil
}

// apEnrichWithFullText fills the item description from the article page body.
func apEnrichWithFullText(ctx context.Context, cl *client.Client, item *models.Item) {
	doc, err := routeutils.GetHTML(ctx, cl, item.Link)
	if err != nil || doc == nil {
		return
	}
	if body := doc.Find("div.RichTextStoryBody").First(); body.Length() > 0 {
		if h, berr := body.Html(); berr == nil && strings.TrimSpace(h) != "" {
			item.Description = h
		}
	}
}

func firstNonEmptyString(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
