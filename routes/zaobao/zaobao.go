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
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

const zaobaoBaseURL = "https://www.zaobao.com.sg"

// zaobaoSections maps route sections to site paths (Singapore edition).
var zaobaoSections = map[string]string{
	"china":     "/news/china",
	"singapore": "/news/singapore",
	"world":     "/news/world",
	"zfinance":  "/finance",
}

var zaobaoRealtimeRoute = routeutils.RouteSpec{
	Path:        "realtime/:section",
	Name:        "Lianhe Zaobao Realtime",
	Example:     "zaobao/realtime/china",
	Maintainers: []string{"xihale"},
	Description: "联合早报 (Lianhe Zaobao Singapore) realtime news. Sections: china (default), singapore, world, zfinance",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("section", "News section: china (default), singapore, world, zfinance; use '-' for china"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  ZaobaoRealtimeHandler,
}

// ZaobaoRealtimeHandler handles /zaobao/realtime/:section
func ZaobaoRealtimeHandler(c *ctxpkg.Context) (*models.Feed, error) {
	section := strings.TrimSpace(c.Param("section"))
	if section == "-" || section == "" {
		section = "china"
	}
	path, ok := zaobaoSections[section]
	if !ok {
		return nil, fmt.Errorf("unknown section %q; supported: china, singapore, world, zfinance", section)
	}
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 15, 30)

	doc, err := routeutils.GetHTML(c.Parent(), c.Client(), zaobaoBaseURL+path)
	if err != nil {
		return nil, err
	}

	// Collect unique story links from listing cards.
	type entry struct {
		link  string
		title string
	}
	var entries []entry
	seen := make(map[string]bool)
	doc.Find(".card-listing .card .content-header a").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok || href == "" {
			return
		}
		if !strings.Contains(href, "/story") && !strings.HasPrefix(href, "http") {
			return
		}
		link, lerr := routeutils.ResolveURL(zaobaoBaseURL, href)
		if lerr != nil {
			return
		}
		if seen[link] {
			return
		}
		title := ""
		if v, exists := s.Attr("aria-label"); exists {
			title = strings.TrimSpace(v)
		}
		if title == "" {
			title = strings.TrimSpace(s.Find("h2").First().Text())
		}
		if title == "" {
			title = strings.TrimSpace(s.Text())
		}
		seen[link] = true
		if title != "" {
			entries = append(entries, entry{link: link, title: title})
		}
	})
	if len(entries) == 0 {
		return nil, fmt.Errorf("no stories found on %s%s", zaobaoBaseURL, path)
	}
	if len(entries) > limit {
		entries = entries[:limit]
	}

	name := zaobaoSectionName(section)
	feed := routeutils.NewFeed(
		fmt.Sprintf("《联合早报》-%s", name),
		zaobaoBaseURL+path,
		"新加坡、中国、亚洲和国际的即时新闻，尽在联合早报。",
	)

	for _, e := range entries {
		item := zaobaoArticleItem(c.Parent(), c.Client(), e.link, e.title)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

func zaobaoSectionName(section string) string {
	switch section {
	case "singapore":
		return "新加坡"
	case "world":
		return "国际"
	case "zfinance":
		return "财经"
	default:
		return "中港台"
	}
}

// zaobaoArticleItem fetches one article page and builds a feed item with
// metadata from its JSON-LD NewsArticle block and body HTML.
func zaobaoArticleItem(ctx context.Context, cl *client.Client, link, fallbackTitle string) *models.Item {
	title := fallbackTitle
	description := ""
	var pub time.Time
	author := ""

	if doc, err := routeutils.GetHTML(ctx, cl, link); err == nil && doc != nil {
		if headline, published, img, a := zaobaoJSONLDMeta(doc); headline != "" {
			title = headline
			pub = published
			author = a
			if img != "" {
				description += fmt.Sprintf(`<img src="%s" alt=""/><br/>`, html.EscapeString(img))
			}
		}
		if body := doc.Find("div.articleBody").First(); body.Length() > 0 {
			if h, berr := body.Html(); berr == nil {
				description += h
			}
		}
	}

	item := routeutils.NewItem(title, link, description, pub)
	item.GUID = link
	if author != "" {
		routeutils.SetAuthor(item, author)
	}
	return item
}

func zaobaoJSONLDMeta(doc *parser.Document) (headline string, published time.Time, image string, author string) {
	var scripts []string
	doc.Find("script[type='application/ld+json']").Each(func(_ int, s *goquery.Selection) {
		scripts = append(scripts, cleanControlChars(s.Text()))
	})
	check := func(meta map[string]any) bool {
		t, _ := meta["@type"].(string)
		return t == "NewsArticle"
	}
	for _, raw := range scripts {
		var meta map[string]any
		if err := json.Unmarshal([]byte(raw), &meta); err != nil {
			continue
		}
		if check(meta) {
			return zaobaoMetaFields(meta)
		}
		// Some pages wrap entities in an @graph array.
		graph, _ := meta["@graph"].([]any)
		for _, g := range graph {
			m, ok := g.(map[string]any)
			if ok && check(m) {
				return zaobaoMetaFields(m)
			}
		}
	}
	return "", time.Time{}, "", ""
}

func zaobaoMetaFields(meta map[string]any) (headline string, published time.Time, image string, author string) {
	headline, _ = meta["headline"].(string)
	if dp, _ := meta["datePublished"].(string); dp != "" {
		if t, err := dateutil.ParseDate(dp); err == nil {
			published = t
		}
	}
	switch img := meta["image"].(type) {
	case []any:
		if len(img) > 0 {
			if m, ok := img[0].(map[string]any); ok {
				image, _ = m["url"].(string)
			}
		}
	case map[string]any:
		image, _ = img["url"].(string)
	case string:
		image = img
	}
	switch a := meta["author"].(type) {
	case map[string]any:
		author, _ = a["name"].(string)
	case []any:
		var names []string
		for _, e := range a {
			if m, ok := e.(map[string]any); ok {
				if n, _ := m["name"].(string); n != "" {
					names = append(names, n)
				}
			}
		}
		author = strings.Join(names, ", ")
	}
	return headline, published, image, author
}

func cleanControlChars(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' || r >= 0x20 {
			b.WriteRune(r)
		}
	}
	return b.String()
}
