package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// juejinAPIBaseURL is the root of the open juejin web APIs.
const juejinAPIBaseURL = "https://api.juejin.cn"

type juejinBookletBaseInfo struct {
	BookletID string `json:"booklet_id"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	CoverImg  string `json:"cover_img"`
	Price     int64  `json:"price"` // 分
	Ctime     int64  `json:"ctime"`
}

type juejinBooklet struct {
	BaseInfo juejinBookletBaseInfo `json:"base_info"`
}

var juejinBooksRoute = routeutils.RouteSpec{
	Path:        "books",
	Name:        "Juejin Books",
	Example:     "juejin/books",
	Maintainers: []string{"xihale"},
	Description: "掘金小册上新（仅更新提醒，不含付费内容）",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     JuejinBooksHandler,
}

// parseJuejinBooks decodes and validates a booklet list envelope.
func parseJuejinBooks(raw json.RawMessage) ([]juejinBooklet, error) {
	var books []juejinBooklet
	if err := json.Unmarshal(raw, &books); err != nil {
		return nil, err
	}
	return books, nil
}

// JuejinBooksHandler handles /juejin/books
func JuejinBooksHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var resp juejinResp
	if err := juejinProfile.PostJSON(juejinAPIBaseURL+"/booklet_api/v1/booklet/listbycategory", map[string]any{
		"category_id": "0",
		"cursor":      "0",
		"limit":       20,
	}).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.ok(); err != nil {
		return nil, err
	}
	books, err := parseJuejinBooks(resp.Data)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("掘金小册", "https://juejin.cn/books", "掘金小册上新提醒（付费内容不作同步）")
	mapJuejinBooklets(feed, books)
	return feed, nil
}

// mapJuejinBooklets converts booklet summaries into feed items.
func mapJuejinBooklets(feed *models.Feed, books []juejinBooklet) {
	for _, book := range books {
		info := book.BaseInfo
		title := strings.TrimSpace(info.Title)
		if title == "" || info.BookletID == "" {
			continue
		}
		var b strings.Builder
		if info.CoverImg != "" {
			b.WriteString(`<img src="` + html.EscapeString(info.CoverImg) + `"/><br>`)
			b.WriteString("<strong>" + html.EscapeString(title) + "</strong><br>")
		}
		if summary := strings.TrimSpace(info.Summary); summary != "" {
			b.WriteString("<p>" + html.EscapeString(summary) + "</p>")
		}
		b.WriteString("<p><strong>价格:</strong> " + html.EscapeString(fmt.Sprintf("%.2f 元", float64(info.Price)/100)) + "</p>")

		pubDate := time.Time{}
		if info.Ctime > 0 {
			pubDate = time.Unix(info.Ctime, 0)
		}
		item := routeutils.NewItem(title, "https://juejin.cn/book/"+info.BookletID, b.String(), pubDate)
		item.GUID = "juejin-booklet-" + info.BookletID
		routeutils.AddItem(feed, item)
	}
}
