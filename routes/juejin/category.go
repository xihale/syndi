package routes

import (
	"fmt"
	"time"

	ctxpkg "github.com/rsshub/go/pkg/context"
	"github.com/rsshub/go/pkg/models"
	"github.com/rsshub/go/pkg/registry"
	"github.com/rsshub/go/internal/routeutils"
)

func init() {
	route := &models.Route{
		Path:         "/juejin/category/:id",
		Name:         "Juejin Category",
		Example:      "juejin/category/1",
		Maintainers:  []string{"yourname"},
		Description:  "Fetch articles from Juejin (Chinese tech community)",
		Categories:   []models.Category{{Name: "programming"}},
		Features:     models.Features{},
		Handler:      JuejinCategoryHandler,
		Parameters: []models.Parameter{
			{Name: "id", Required: true, Description: "Category ID (1: frontend, 2: backend, 3: Android, 4: iOS, 5: AI, 6: tools)"},
		},
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// JuejinCategoryHandler handles /juejin/category/:id
func JuejinCategoryHandler(c *ctxpkg.Context) (*models.Feed, error) {
	categoryID := c.Param("id")
	ctx := c.Parent()

	url := fmt.Sprintf("https://api.juejin.cn/recommend_api/v1/article/recommend_all_feed?aid=2608&uuid=&id=%s&count=20", categoryID)

	var response JuejinResponse
	if err := routeutils.GetJSON(ctx, c.Client(), url, &response); err != nil {
		return nil, err
	}

	if response.ErrNo != 0 {
		return nil, fmt.Errorf("juejin API error: %s", response.ErrMsg)
	}

	categoryNames := map[string]string{
		"1":  "Frontend",
		"2":  "Backend",
		"3":  "Android",
		"4":  "iOS",
		"5":  "AI",
		"6":  "Tools",
	}

	categoryName := categoryNames[categoryID]
	if categoryName == "" {
		categoryName = "Category " + categoryID
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Juejin %s", categoryName),
		fmt.Sprintf("https://juejin.cn/%s", categoryID),
		fmt.Sprintf("Articles from Juejin %s category", categoryName),
	)

	for _, item := range response.Data {
		if item.ArticleInfo.ID == "" {
			continue
		}

		// Build description
		description := item.ArticleInfo.Content
		if item.ArticleInfo.CoverImage != "" {
			description = fmt.Sprintf("<img src=\"%s\"/><br/>%s", item.ArticleInfo.CoverImage, description)
		}

		pubDate := time.Unix(item.ArticleInfo.Ctime, 0)

		feedItem := routeutils.NewItem(
			item.ArticleInfo.Title,
			fmt.Sprintf("https://juejin.cn/post/%s", item.ArticleInfo.ID),
			description,
			pubDate,
		)
		feedItem.GUID = "juejin-" + item.ArticleInfo.ID

		// Set author
		if item.AuthorUserInfo != nil && item.AuthorUserInfo.UserName != "" {
			routeutils.SetAuthor(feedItem, item.AuthorUserInfo.UserName,
				routeutils.WithAuthorURI(fmt.Sprintf("https://juejin.cn/user/%s", item.AuthorUserInfo.UserID)))
		}

		// Add categories
		routeutils.SetCategories(feedItem, categoryName)
		if item.ArticleInfo.Category != "" {
			routeutils.SetCategories(feedItem, item.ArticleInfo.Category)
		}

		routeutils.AddItem(feed, feedItem)
	}

	return feed, nil
}

type JuejinResponse struct {
	ErrNo  int           `json:"err_no"`
	ErrMsg string        `json:"err_msg"`
	Data   []JuejinItem  `json:"data"`
}

type JuejinItem struct {
	ArticleInfo     JuejinArticleInfo `json:"article_info"`
	AuthorUserInfo  *JuejinAuthor     `json:"author_user_info"`
}

type JuejinArticleInfo struct {
	ID         string `json:"article_id"`
	Title      string `json:"title"`
	Content    string `json:"brief_content"`
	CoverImage string `json:"cover_image"`
	Ctime      int64  `json:"ctime"`
	Mtime      int64  `json:"mtime"`
	Category   string `json:"category"`
	CategoryID string `json:"category_id"`
	Tags       string `json:"tags"`
}

type JuejinAuthor struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}
