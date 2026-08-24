package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// krProfile disguises requests against 36kr.com pages.
var krProfile = disguise.Chrome().Lang("zh-CN,zh;q=0.9")

// krItemListRe extracts the embedded itemList JSON array from SSR pages.
var krItemListRe = regexp.MustCompile(`(?s)"itemList":(\[.*?\])`)

// krID accepts itemId as either a JSON string or number.
type krID string

func (s *krID) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) >= 2 && b[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*s = krID(v)
		return nil
	}
	*s = krID(b)
	return nil
}

type krItem struct {
	ItemID           krID   `json:"itemId"`
	ItemType         int    `json:"itemType"`
	WidgetTitle      string `json:"widgetTitle"`
	WidgetContent    string `json:"widgetContent"`
	Summary          string `json:"summary"`
	PublishTime      int64  `json:"publishTime"` // unix milliseconds
	AuthorName       string `json:"authorName"`
	TemplateMaterial *struct {
		ItemID        krID   `json:"itemId"`
		WidgetTitle   string `json:"widgetTitle"`
		WidgetContent string `json:"widgetContent"`
		Summary       string `json:"summary"`
		PublishTime   int64  `json:"publishTime"`
		AuthorName    string `json:"authorName"`
	} `json:"templateMaterial"`
}

func (i krItem) material() krItem {
	if i.TemplateMaterial != nil {
		m := i.TemplateMaterial
		return krItem{
			ItemID:        m.ItemID,
			WidgetTitle:   m.WidgetTitle,
			WidgetContent: m.WidgetContent,
			Summary:       m.Summary,
			PublishTime:   m.PublishTime,
			AuthorName:    m.AuthorName,
		}
	}
	return i
}

func (i krItem) title() string {
	t := i.WidgetTitle
	t = strings.ReplaceAll(t, "<em>", "")
	t = strings.ReplaceAll(t, "</em>", "")
	return html.UnescapeString(strings.TrimSpace(t))
}

// parseKrItems extracts and maps the itemList payload into feed items.
func parseKrItems(pageHTML, linkBase string, limit int) ([]*models.Item, error) {
	m := krItemListRe.FindStringSubmatch(pageHTML)
	if m == nil {
		return nil, fmt.Errorf("36kr: itemList not found in page")
	}
	var raw []krItem
	if err := json.Unmarshal([]byte(m[1]), &raw); err != nil {
		return nil, fmt.Errorf("36kr: bad itemList JSON: %w", err)
	}
	items := make([]*models.Item, 0, len(raw))
	for _, r := range raw {
		if r.ItemType == 0 && r.TemplateMaterial == nil {
			continue
		}
		it := r.material()
		title := it.title()
		if title == "" || it.ItemID == "" || len(items) >= limit {
			continue
		}
		desc := ""
		if s := strings.TrimSpace(it.WidgetContent); s != "" {
			desc = "<p>" + html.EscapeString(s) + "</p>"
		} else if s := strings.TrimSpace(it.Summary); s != "" {
			desc = "<p>" + html.EscapeString(s) + "</p>"
		}
		var pubDate time.Time
		if it.PublishTime > 0 {
			pubDate = time.UnixMilli(it.PublishTime)
		}
		item := routeutils.NewItem(title, linkBase+"/"+string(it.ItemID), desc, pubDate)
		item.GUID = string(it.ItemID)
		routeutils.SetItemAuthor(item, it.AuthorName, "", "")
		items = append(items, item)
	}
	return items, nil
}

var krNewsflashesRoute = routeutils.RouteSpec{
	Path:        "newsflashes",
	Name:        "36Kr Newsflashes",
	Example:     "36kr/newsflashes",
	Maintainers: []string{"xihale"},
	Description: "Latest 36kr newsflashes (快讯)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     KrNewsflashesHandler,
}

// KrNewsflashesHandler handles /36kr/newsflashes
func KrNewsflashesHandler(c *ctxpkg.Context) (*models.Feed, error) {
	pageURL := "https://www.36kr.com/newsflashes"
	pageHTML, err := krProfile.Fetch(pageURL).GetString(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}
	feed := routeutils.NewFeed("36氪 - 快讯", pageURL, "36氪快讯")
	items, err := parseKrItems(pageHTML, "https://www.36kr.com/newsflashes", 30)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		routeutils.AddItem(feed, it)
	}
	return feed, nil
}

var krInformationRoute = routeutils.RouteSpec{
	Path:        "information/web_news",
	Name:        "36Kr Latest News",
	Example:     "36kr/information/web_news",
	Maintainers: []string{"xihale"},
	Description: "Latest 36kr news channel articles",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "default 20, max 30"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  KrInformationHandler,
}

// KrInformationHandler handles /36kr/information/web_news
func KrInformationHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 30)
	pageURL := "https://www.36kr.com/information/web_news/"
	pageHTML, err := krProfile.Fetch(pageURL).GetString(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}
	feed := routeutils.NewFeed("36氪 - 资讯", pageURL, "36氪最新资讯")
	items, err := parseKrItems(pageHTML, "https://www.36kr.com/p", limit)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		routeutils.AddItem(feed, it)
	}
	return feed, nil
}
