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

// juejinCollection is the interact_api/v1/collectionSet/get payload.
type juejinCollection struct {
	Detail struct {
		TagName string `json:"tag_name"`
	} `json:"detail"`
	CreateUser  juejinAuthorInfo     `json:"create_user"`
	ArticleList []juejinArticleEntry `json:"article_list"`
}

var juejinCollectionRoute = routeutils.RouteSpec{
	Path:        "collection/:collectionId",
	Name:        "Juejin Collection",
	Example:     "juejin/collection/6845242107762835464",
	Maintainers: []string{"xihale"},
	Description: "掘金用户单个收藏夹文章",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("collectionId", "收藏夹 id，可在收藏夹页 URL 中找到"),
	},
	CacheTTL: time.Hour,
	Handler:  JuejinCollectionHandler,
}

// parseJuejinCollection decodes and validates a collection set envelope.
func parseJuejinCollection(raw json.RawMessage) (*juejinCollection, error) {
	var col juejinCollection
	if err := json.Unmarshal(raw, &col); err != nil {
		return nil, err
	}
	return &col, nil
}

// JuejinCollectionHandler handles /juejin/collection/:collectionId
func JuejinCollectionHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("collectionId")

	var resp juejinResp
	if err := juejinProfile.Fetch(fmt.Sprintf("%s/interact_api/v1/collectionSet/get?tag_id=%s&cursor=0", juejinAPIBaseURL, id)).
		GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.ok(); err != nil {
		return nil, err
	}
	col, err := parseJuejinCollection(resp.Data)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(col.Detail.TagName)
	if title == "" {
		title = "收藏夹 " + id
	}
	feedTitle := title + "的收藏集 - 掘金"
	if user := strings.TrimSpace(col.CreateUser.UserName); user != "" {
		feedTitle = fmt.Sprintf("%s - %s的收藏集 - 掘金", title, user)
	}
	feed := routeutils.NewFeed(feedTitle, "https://juejin.cn/collection/"+id, "掘金，用户单个收藏夹")
	mapJuejinEntries(feed, col.ArticleList, "juejin-collection-")
	return feed, nil
}
