package routes

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// sspaiListTopics is the /api/v1/topics payload.
type sspaiListTopics struct {
	List []sspaiTopic `json:"list"`
}

var sspaiTopicsRoute = routeutils.RouteSpec{
	Path:        "topics",
	Name:        "sspai Topics",
	Example:     "sspai/topics",
	Maintainers: []string{"xihale"},
	Description: "少数派专题广场上新（集合页而非单篇文章）",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     SspaiTopicsHandler,
}

// SspaiTopicsHandler handles /sspai/topics
func SspaiTopicsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	endpoint := sspaiAPIPrefix + "/topics?offset=0&limit=20"
	var resp sspaiListTopics
	if err := sspaiAPIProfile().Referer(sspaiBaseURL+"/").Fetch(endpoint).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("少数派 -- 最新专题", sspaiBaseURL+"/topics", "少数派新上线专题（集合页而非具体文章）")
	mapSspaiTopics(feed, resp.List)
	return feed, nil
}

// mapSspaiTopics converts topic entries into feed items.
func mapSspaiTopics(feed *models.Feed, topics []sspaiTopic) {
	for _, topic := range topics {
		title := strings.TrimSpace(topic.Title)
		if title == "" || topic.ID == 0 {
			continue
		}
		var b strings.Builder
		if topic.Banner != "" {
			b.WriteString(`<img src="` + html.EscapeString(sspaiCDNPrefix+topic.Banner) + `"/><br>`)
		}
		if intro := strings.TrimSpace(topic.Intro); intro != "" {
			b.WriteString("<p>" + html.EscapeString(intro) + "</p>")
		}
		pubDate := time.Time{}
		if topic.ReleasedAt > 0 {
			pubDate = time.Unix(topic.ReleasedAt, 0)
		}
		link := fmt.Sprintf("%s/topic/%d", sspaiBaseURL, topic.ID)
		item := routeutils.NewItem(title, link, b.String(), pubDate)
		item.GUID = "sspai-topic-" + strconv.FormatInt(topic.ID, 10)
		routeutils.SetItemAuthor(item, topic.Author.Nickname, "", "")
		routeutils.AddItem(feed, item)
	}
}

var sspaiBookmarksRoute = routeutils.RouteSpec{
	Path:        "bookmarks/:slug",
	Name:        "sspai Bookmarks",
	Example:     "sspai/bookmarks/urfp0d9i",
	Maintainers: []string{"xihale"},
	Description: "少数派用户公开收藏",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("slug", "用户 slug，可在个人主页 URL 中找到"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  SspaiBookmarksHandler,
}

// SspaiBookmarksHandler handles /sspai/bookmarks/:slug
func SspaiBookmarksHandler(c *ctxpkg.Context) (*models.Feed, error) {
	slug := c.Param("slug")
	referer := fmt.Sprintf("%s/u/%s/bookmark_posts", sspaiBaseURL, url.PathEscape(slug))

	endpoint := fmt.Sprintf("%s/article/user/favorite/public/page/get?limit=10&offset=0&slug=%s&type=all", sspaiAPIPrefix, url.QueryEscape(slug))
	favResp, err := fetchSspaiWrapped[[]sspaiArticle](c, endpoint, referer)
	if err != nil {
		return nil, err
	}

	userResp, err := fetchSspaiWrapped[sspaiSlugInfo](c, sspaiAPIPrefix+"/user/slug/info/get?slug="+url.QueryEscape(slug), referer)
	if err != nil || userResp.Data.Nickname == "" {
		userResp = &sspaiWrapped[sspaiSlugInfo]{Data: sspaiSlugInfo{Nickname: slug}}
	}

	feed := routeutils.NewFeed(
		userResp.Data.Nickname+" 的全部收藏 - 少数派",
		referer,
		fmt.Sprintf("少数派用户「%s」的全部收藏", userResp.Data.Nickname),
	)
	mapSspaiArticles(feed, favResp.Data, "sspai-bookmark-")
	return feed, nil
}
