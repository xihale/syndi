package routes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// juejinTagDetail is the tag_api/v1/query_tag_detail payload.
type juejinTagDetail struct {
	TagID string        `json:"tag_id"`
	Icon  string        `json:"icon"` // nested convenience; real icon lives in Tag
	Tag   juejinTagFull `json:"tag"`
}

type juejinTagFull struct {
	TagID   string `json:"tag_id"`
	TagName string `json:"tag_name"`
	Icon    string `json:"icon"`
}

// icon returns the tag icon from whichever level the API populated.
func (d *juejinTagDetail) tagIcon() string {
	if d.Tag.Icon != "" {
		return d.Tag.Icon
	}
	return d.Icon
}

var juejinTagRoute = routeutils.RouteSpec{
	Path:        "tag/:tag",
	Name:        "Juejin Tag",
	Example:     "juejin/tag/JavaScript",
	Maintainers: []string{"xihale"},
	Description: "掘金标签下最热文章",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("tag", "标签名，可在标签 URL 中找到"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  JuejinTagHandler,
}

// parseJuejinTagDetail decodes and validates a tag detail envelope.
func parseJuejinTagDetail(raw json.RawMessage) (*juejinTagDetail, error) {
	var detail juejinTagDetail
	if err := json.Unmarshal(raw, &detail); err != nil {
		return nil, err
	}
	if detail.TagID == "" {
		detail.TagID = detail.Tag.TagID
	}
	if detail.TagID == "" {
		return nil, fmt.Errorf("juejin: tag id missing in query_tag_detail response")
	}
	return &detail, nil
}

// JuejinTagHandler handles /juejin/tag/:tag
func JuejinTagHandler(c *ctxpkg.Context) (*models.Feed, error) {
	tag := c.Param("tag")

	var resp juejinResp
	if err := juejinProfile.PostJSON(juejinAPIBaseURL+"/tag_api/v1/query_tag_detail", map[string]any{
		"key_word": tag,
	}).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.ok(); err != nil {
		return nil, err
	}
	detail, err := parseJuejinTagDetail(resp.Data)
	if err != nil {
		return nil, err
	}

	entries, err := postJuejinArticles(c,
		juejinAPIBaseURL+"/recommend_api/v1/article/recommend_tag_feed",
		map[string]any{
			"id_type":   2,
			"cursor":    "0",
			"tag_ids":   []string{detail.TagID},
			"sort_type": 300,
		}, false)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeedWithOptions(routeutils.FeedOptions{
		Title:       "掘金 " + tag,
		Link:        "https://juejin.cn/tag/" + url.PathEscape(tag),
		Description: "掘金 " + tag + " 标签下热门文章",
		Image:       detail.tagIcon(),
	})
	mapJuejinEntries(feed, entries, "juejin-tag-")
	return feed, nil
}
