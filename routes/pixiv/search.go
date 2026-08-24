package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// pixivSearchItem is one entry of the search result lists. Both artwork and
// novel results share most fields; novels add textCount/genre.
type pixivSearchItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"` // i.pximg.net thumbnail, hotlink-blocked
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	UserID      string    `json:"userId"`
	UserName    string    `json:"userName"`
	PageCount   int       `json:"pageCount"`
	TextCount   int       `json:"textCount"`
	CreateDate  pixivTime `json:"createDate"`
	UpdateDate  pixivTime `json:"updateDate"`
	XRestrict   int       `json:"xRestrict"`
	AiType      int       `json:"aiType"` // 2 = AI-generated
}

// PixivSearchHandler serves /pixiv/search/:keyword — newest artwork search.
func PixivSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	if err := requirePixivCookies(); err != nil {
		return nil, err
	}
	return pixivSearchFeed(c, "artworks")
}

// PixivNovelSearchHandler serves /pixiv/novel-search/:keyword — newest novel
// search. Kept as a separate static prefix because gin cannot mix ":keyword"
// with a longer static child under /search.
func PixivNovelSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	if err := requirePixivCookies(); err != nil {
		return nil, err
	}
	return pixivSearchFeed(c, "novels")
}

func pixivSearchFeed(c *ctxpkg.Context, kind string) (*models.Feed, error) {
	keyword := c.Param("keyword")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 60)
	ctx := c.Parent()

	apiURL := fmt.Sprintf("%s/ajax/search/%s/%s?%s",
		pixivBaseURL, kind, url.PathEscape(keyword), url.Values{
			"word":     {keyword},
			"order":    {"date_d"},
			"all_mode": {"s"},
			"s_mode":   {"s_tag"},
			"type":     {"all"},
			"p":        {"1"},
			"lang":     {"en"},
		}.Encode())

	var resp pixivAjaxResp
	if err := pixivProfile(pixivReferer).Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Error {
		return nil, fmt.Errorf("pixiv: search rejected the request (%s)", resp.Message)
	}

	isNovel := kind == "novels"
	var body struct {
		Data []pixivSearchItem `json:"data"`
	}
	// The result list sits under different body keys: "illustManga" for
	// artwork searches and "novel" for novel searches.
	section := "illustManga"
	if isNovel {
		section = "novel"
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(resp.Body, &wrapper); err != nil {
		return nil, fmt.Errorf("pixiv: unexpected search payload: %w", err)
	}
	rawList, ok := wrapper[section]
	if !ok {
		return nil, fmt.Errorf("pixiv: search payload missing %q section", section)
	}
	if err := json.Unmarshal(rawList, &body); err != nil && len(body.Data) == 0 {
		return nil, fmt.Errorf("pixiv: unexpected %s payload: %w", kind, err)
	}

	feedTitle := fmt.Sprintf("%s 的 pixiv 插画搜索", keyword)
	feedDesc := fmt.Sprintf("pixiv 上关于 %s 的最新插画", keyword)
	tagPath := "artworks"
	if isNovel {
		feedTitle = fmt.Sprintf("%s 的 pixiv 小说搜索", keyword)
		feedDesc = fmt.Sprintf("pixiv 上关于 %s 的最新小说", keyword)
		tagPath = "novels"
	}
	feed := routeutils.NewFeed(feedTitle, fmt.Sprintf("%s/tags/%s/%s", pixivBaseURL, url.PathEscape(keyword), tagPath), feedDesc)

	for _, it := range body.Data {
		title := strings.TrimSpace(it.Title)
		if title == "" || it.ID == "" {
			continue
		}
		if len(feed.Items) >= limit {
			break
		}

		link := fmt.Sprintf("%s/artworks/%s", pixivBaseURL, it.ID)
		prefix := ""
		if it.AiType == 2 {
			prefix = "[AI] "
		}
		if it.XRestrict > 0 {
			prefix += "[R-18] "
		}

		var b strings.Builder
		if it.Description != "" {
			b.WriteString(it.Description)
			b.WriteString("<br/>")
		}
		fmt.Fprintf(&b, "<p>画师：%s</p>", html.EscapeString(it.UserName))
		if isNovel {
			fmt.Fprintf(&b, "<p>字数：%d</p>", it.TextCount)
		} else if it.PageCount > 1 {
			fmt.Fprintf(&b, "<p>页数：%d</p>", it.PageCount)
		}
		if !isNovel {
			fmt.Fprintf(&b, `<p><img src="%s"/></p>`, html.EscapeString(pixivEmbedImageURL(it.ID)))
		}

		item := routeutils.NewItem(prefix+title, link, b.String(), it.CreateDate.Time)
		if item == nil {
			continue
		}
		item.GUID = "pixiv-" + kind + "-" + it.ID
		routeutils.SetItemAuthor(item, it.UserName, "", fmt.Sprintf("%s/users/%s", pixivBaseURL, it.UserID))
		routeutils.SetCategories(item, it.Tags...)
		routeutils.AddItem(feed, item)
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("pixiv: no %s results for keyword %q (cookies may be expired)", kind, keyword)
	}
	return feed, nil
}
