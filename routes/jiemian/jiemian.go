package routes

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const jiemianRootURL = "https://www.jiemian.com"

// jiemianProfile disguises requests against jiemian.com pages.
var jiemianProfile = disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer(jiemianRootURL + "/")

// jiemianArticleLinkRe matches article/video detail links on list pages.
var jiemianArticleLinkRe = regexp.MustCompile(`/(article|video)/\w+\.html`)

// --- Route specs ---

var jiemianHomeRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Jiemian News",
	Example:     "jiemian",
	Maintainers: []string{"xihale"},
	Description: "Latest news from the Jiemian News homepage (界面新闻首页)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "item count, default 20, max 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  JiemianHomeHandler,
}

var jiemianListsRoute = routeutils.RouteSpec{
	Path:        "lists/:id",
	Name:        "Jiemian Column",
	Example:     "jiemian/lists/4",
	Maintainers: []string{"xihale"},
	Description: "Articles from a Jiemian column. IDs: business=2, finance=800, news=801, culture=130, express=4, tech=65, finance deep=9, securities=112, real estate=62, auto=51, health=472",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "column ID, see description"),
		routeutils.OptionalParam("limit", "item count, default 20, max 50"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  JiemianListsHandler,
}

// Routes lists all jiemian route specs in this package.
var Routes = []routeutils.RouteSpec{
	jiemianHomeRoute,
	jiemianListsRoute,
}

// --- Handlers ---

// JiemianHomeHandler handles /jiemian.
func JiemianHomeHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return jiemianCategoryFeed(c, "")
}

// JiemianListsHandler handles /jiemian/lists/:id.
func JiemianListsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return jiemianCategoryFeed(c, "lists/"+c.Param("id"))
}

type jiemianEntry struct {
	link  string
	title string
}

// jiemianCategoryFeed scrapes a Jiemian list page and enriches entries with
// article details.
func jiemianCategoryFeed(c *ctxpkg.Context, category string) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	currentURL := jiemianRootURL + "/"
	if category != "" {
		currentURL = fmt.Sprintf("%s/%s.html", jiemianRootURL, category)
	}

	doc, err := jiemianProfile.Fetch(currentURL).GetHTML(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var list []jiemianEntry
	doc.Each("a", func(_ int, s *parser.Selection) {
		href := s.AttrOr("href", "")
		if href == "" || !jiemianArticleLinkRe.MatchString(href) {
			return
		}
		link := href
		if strings.HasPrefix(href, "/") {
			link = jiemianRootURL + href
		} else if !strings.HasPrefix(href, "http") {
			return
		}
		if seen[link] {
			return
		}
		seen[link] = true
		list = append(list, jiemianEntry{link: link, title: strings.TrimSpace(s.Text())})
	})
	if len(list) > limit {
		list = list[:limit]
	}

	feed := routeutils.NewFeed(jiemianFeedTitle(doc), currentURL, jiemianMetaDescription(doc))

	items := make([]*models.Item, len(list))
	var sem = make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, entry := range list {
		wg.Add(1)
		go func(idx int, entry jiemianEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			items[idx] = jiemianBuildItem(c, entry)
		}(i, entry)
	}
	wg.Wait()
	for _, item := range items {
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

// jiemianBuildItem fetches one article page and builds the feed item; falls
// back to the anchor text when the detail page cannot be fetched.
func jiemianBuildItem(c *ctxpkg.Context, entry jiemianEntry) *models.Item {
	item := &models.Item{Title: entry.title, Link: entry.link, GUID: entry.link}

	page, err := jiemianProfile.Fetch(entry.link).GetString(c.Parent(), c.Client())
	if err != nil {
		if entry.title != "" {
			return item
		}
		return nil
	}
	detail, err := parser.LoadString(page)
	if err != nil {
		if entry.title != "" {
			return item
		}
		return nil
	}

	title := strings.TrimSpace(detail.Text("div.article-header h1"))
	if title == "" {
		title = entry.title
	}
	if title == "" {
		return nil
	}
	item.Title = title

	intro := strings.TrimSpace(detail.Text("div.article-header p"))
	contentHTML, _ := detail.Find("div.article-content").First().Html()

	desc := ""
	image := detail.First("div.article-img img")
	if image != nil && image.Length() > 0 {
		src := image.AttrOr("src", "")
		alt := image.Next().Text()
		if alt == "" {
			alt = title
		}
		desc += `<figure><img src="` + html.EscapeString(src) + `" alt="` + html.EscapeString(strings.TrimSpace(alt)) + `"/></figure>`
	}
	if intro != "" {
		desc += "<p>" + html.EscapeString(intro) + "</p>"
	}
	desc += contentHTML
	item.Description = desc

	var authors []string
	if author := detail.First("span.author"); author != nil {
		author.Find("a").Each(func(_ int, a *parser.Selection) {
			if name := strings.TrimSpace(a.Text()); name != "" {
				authors = append(authors, name)
			}
		})
	}
	if len(authors) > 0 {
		routeutils.SetItemAuthor(item, strings.Join(authors, "/"), "", "")
	}

	var categories []string
	detail.Each("meta.meta-container a", func(_ int, cat *parser.Selection) {
		if name := strings.TrimSpace(cat.Text()); name != "" {
			categories = append(categories, name)
		}
	})
	routeutils.SetCategories(item, categories...)

	if tsRaw, ok := detail.Attr("span[data-article-publish-time]", "data-article-publish-time"); ok {
		if ts, err := strconv.ParseInt(tsRaw, 10, 64); err == nil && ts > 0 {
			item.PubDate = time.Unix(ts, 0)
		}
	}
	return item
}

// jiemianFeedTitle derives "<column> | 界面新闻" style titles.
func jiemianFeedTitle(doc *parser.Document) string {
	t := strings.TrimSpace(doc.Text("title"))
	if t == "" {
		return "界面新闻"
	}
	parts := strings.Split(t, "_")
	if len(parts) > 1 && parts[0] != "" {
		return parts[0] + "_" + parts[len(parts)-1]
	}
	return t
}

func jiemianMetaDescription(doc *parser.Document) string {
	if d, ok := doc.Attr(`meta[name="description"]`, "content"); ok {
		return d
	}
	return "界面新闻"
}
