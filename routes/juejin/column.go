package routes

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// juejinColumnDetail is the content_api/v1/column/detail payload.
type juejinColumnDetail struct {
	Author        juejinAuthorInfo `json:"author"`
	ColumnVersion struct {
		Title string `json:"title"`
		Intro string `json:"description"` // legacy field name
		Text  string `json:"content"`     // current intro location
		Cover string `json:"cover"`
	} `json:"column_version"`
}

// intro returns whichever column description field the API populated.
func (d *juejinColumnDetail) intro() string {
	if s := strings.TrimSpace(d.ColumnVersion.Intro); s != "" {
		return s
	}
	return strings.TrimSpace(d.ColumnVersion.Text)
}

var juejinColumnRoute = routeutils.RouteSpec{
	Path:        "column/:id",
	Name:        "Juejin Column",
	Example:     "juejin/column/6960559453037199391",
	Maintainers: []string{"xihale"},
	Description: "掘金专栏最新文章",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "专栏 id，可在专栏页 URL 中找到"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  JuejinColumnHandler,
}

// parseJuejinColumnDetail decodes and validates a column detail envelope.
func parseJuejinColumnDetail(raw json.RawMessage) (*juejinColumnDetail, error) {
	var detail juejinColumnDetail
	if err := json.Unmarshal(raw, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// JuejinColumnHandler handles /juejin/column/:id
func JuejinColumnHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")

	var colResp juejinResp
	endpoint := fmt.Sprintf("%s/content_api/v1/column/detail?column_id=%s", juejinAPIBaseURL, id)
	if err := juejinProfile.Fetch(endpoint).GetJSON(c.Parent(), c.Client(), &colResp); err != nil {
		return nil, err
	}
	if err := colResp.ok(); err != nil {
		return nil, err
	}
	detail, err := parseJuejinColumnDetail(colResp.Data)
	if err != nil {
		return nil, err
	}

	entries, err := postJuejinArticles(c,
		juejinAPIBaseURL+"/content_api/v1/column/articles_cursor",
		map[string]any{
			"column_id": id,
			"cursor":    "0",
			"limit":     20,
			"sort":      0,
		}, false)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(detail.ColumnVersion.Title)
	if title == "" {
		title = "专栏 " + id
	}
	feedTitle := title + "的专栏 - 掘金"
	if author := strings.TrimSpace(detail.Author.UserName); author != "" {
		feedTitle = fmt.Sprintf("%s - %s的专栏 - 掘金", title, author)
	}
	feed := routeutils.NewFeedWithOptions(routeutils.FeedOptions{
		Title:       feedTitle,
		Link:        "https://juejin.cn/column/" + id,
		Description: detail.intro(),
		Image:       detail.ColumnVersion.Cover,
	})
	mapJuejinEntries(feed, entries, "juejin-column-")
	return feed, nil
}
