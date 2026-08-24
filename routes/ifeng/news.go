package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const ifengRootURL = "https://news.ifeng.com"

func ifengProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer(ifengRootURL + "/")
}

type ifengStreamItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	NewsTime string `json:"newsTime"`
	Source   string `json:"source"`
	Thumbs   struct {
		Image []struct {
			URL string `json:"url"`
		} `json:"image"`
	} `json:"thumbnails"`
}

var (
	ifengStreamRe = regexp.MustCompile(`(?s)"newsstream":(\[.*?\]),"cooperation"`)
	ifengEditorRe = regexp.MustCompile(`"editorName":"(.*?)"`)
	ifengListRe   = regexp.MustCompile(`(?s)"contentList":(\[.*?\])`)
)

var ifengNewsRoute = routeutils.RouteSpec{
	Path:        "news",
	Name:        "Ifeng News Headlines",
	Example:     "ifeng/news",
	Maintainers: []string{"xihale"},
	Description: "Phoenix News (凤凰网) homepage rolling headlines with full text",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Max items, default 15"),
		routeutils.OptionalParam("fulltext", "Fetch full article text, default true; set 0 to disable"),
	},
	CacheTTL: 20 * time.Minute,
	Handler:  IfengNewsHandler,
}

// IfengNewsHandler handles /ifeng/news
func IfengNewsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 15, 40)
	fulltext := routeutils.ParseBool(c.QueryParam("fulltext"), true)
	ctx := c.Parent()

	body, err := ifengProfile().Fetch(ifengRootURL+"/").GetString(ctx, c.Client())
	if err != nil {
		return nil, err
	}
	m := ifengStreamRe.FindStringSubmatch(body)
	if m == nil {
		return nil, fmt.Errorf("ifeng: newsstream block not found on homepage")
	}
	var stream []ifengStreamItem
	if err := json.Unmarshal([]byte(m[1]), &stream); err != nil {
		return nil, fmt.Errorf("ifeng: parse newsstream: %w", err)
	}

	feed := routeutils.NewFeed(
		"凤凰网-资讯",
		ifengRootURL+"/",
		"凤凰网首页要闻滚动",
	)
	n := 0
	for _, s := range stream {
		if n >= limit {
			break
		}
		if s.Title == "" || s.URL == "" {
			continue
		}
		desc := "<p>" + html.EscapeString(s.Title) + "</p>"
		if len(s.Thumbs.Image) > 0 && s.Thumbs.Image[len(s.Thumbs.Image)-1].URL != "" {
			desc = `<img src="` + html.EscapeString(s.Thumbs.Image[len(s.Thumbs.Image)-1].URL) + `"/><br/>` + desc
		}
		author := s.Source
		if fulltext {
			d, a := ifengFetchArticle(ctx, c.Client(), s.URL)
			if d != "" {
				desc = d
			}
			if a != "" {
				author = a
			}
		}
		item := routeutils.NewItem(s.Title, s.URL, desc, ifengParseTime(s.NewsTime))
		if item == nil {
			continue
		}
		if s.ID != "" {
			item.GUID = s.ID
		}
		if author != "" {
			routeutils.SetItemAuthor(item, author, "", "")
		}
		routeutils.AddItem(feed, item)
		n++
	}
	return feed, nil
}

// ifengFetchArticle extracts the article body and editor from an article page.
func ifengFetchArticle(ctx context.Context, cl *client.Client, pageURL string) (desc, author string) {
	body, err := ifengProfile().Referer(pageURL).Fetch(pageURL).GetString(ctx, cl)
	if err != nil {
		return "", ""
	}
	if mm := ifengEditorRe.FindStringSubmatch(body); mm != nil {
		author = mm[1]
	}
	lm := ifengListRe.FindStringSubmatch(body)
	if lm == nil {
		return "", author
	}
	var entries []json.RawMessage
	if json.Unmarshal([]byte(lm[1]), &entries) != nil {
		return "", author
	}
	var b strings.Builder
	for _, raw := range entries {
		var text string
		if json.Unmarshal(raw, &text) == nil {
			b.WriteString(text)
		}
	}
	content := strings.TrimSpace(b.String())
	if content == "" {
		return "", author
	}
	return content, author
}

func ifengParseTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, ifengCST)
	if err != nil {
		return time.Time{}
	}
	return t
}

var ifengCST = time.FixedZone("CST", 8*60*60)
