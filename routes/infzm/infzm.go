package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const (
	infzmBaseURL    = "https://www.infzm.com"
	infzmCSTOffset  = 8 * 60 * 60
	infzMTimeLayout = "2006-01-02 15:04:05"
)

var infzmLocation = time.FixedZone("CST", infzmCSTOffset)

func infzmProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer(infzmBaseURL + "/")
}

type infzmArticle struct {
	ID          int64  `json:"id"`
	Subject     string `json:"subject"`
	Author      string `json:"author"`
	PublishTime string `json:"publish_time"`
	Introtext   string `json:"introtext"`
}

type infzmContentsResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Contents    []infzmArticle `json:"contents"`
		HotContents []infzmArticle `json:"hot_contents"`
		CurrentTerm struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
		} `json:"current_term"`
	} `json:"data"`
}

var infzmHotRoute = routeutils.RouteSpec{
	Path:        "hot",
	Name:        "Infzm Hot Articles",
	Example:     "infzm/hot",
	Maintainers: []string{"xihale"},
	Description: "Southern Weekly (南方周末) most-read articles",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     InfzmHotHandler,
}

// InfzmHotHandler handles /infzm/hot
func InfzmHotHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var resp infzmContentsResp
	if err := infzmProfile().JSONAccept().
		Fetch(infzmBaseURL+"/hot_contents").
		GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("infzm api code %d: %s", resp.Code, resp.Msg)
	}

	feed := routeutils.NewFeed(
		"南方周末-热门文章",
		infzmBaseURL+"/",
		"南方周末热门文章",
	)
	infzmAppendArticles(feed, c, resp.Data.HotContents)
	return feed, nil
}

// infzmChannelIDs lists the well-known channel term ids.
var infzmChannelNames = map[string]string{
	"1": "推荐", "2": "新闻", "3": "观点", "4": "文化", "5": "生活",
	"6": "专题", "7": "人物", "8": "影像", "131": "视频",
}

var infzmChannelRoute = routeutils.RouteSpec{
	Path:        "channel/:id",
	Name:        "Infzm Channel",
	Example:     "infzm/channel/2",
	Maintainers: []string{"xihale"},
	Description: "Southern Weekly (南方周末) channel feed. Ids: 1 推荐, 2 新闻, 3 观点, 4 文化, 5 生活, 6 专题, 7 人物, 8 影像, 131 视频",
	Categories:  []models.Category{{Name: "new-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Channel term id, e.g. 2 for news"),
		routeutils.OptionalParam("limit", "Max items, default 20"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  InfzmChannelHandler,
}

// InfzmChannelHandler handles /infzm/channel/:id
func InfzmChannelHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)

	apiURL := fmt.Sprintf("%s/contents?term_id=%s&page=1&format=json", infzmBaseURL, id)
	var resp infzmContentsResp
	if err := infzmProfile().Referer(fmt.Sprintf("%s/contents?term_id=%s", infzmBaseURL, id)).
		JSONAccept().Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("infzm api code %d: %s", resp.Code, resp.Msg)
	}

	name := resp.Data.CurrentTerm.Title
	if name == "" {
		name = infzmChannelNames[id]
	}
	feed := routeutils.NewFeed(
		"南方周末-"+name,
		fmt.Sprintf("%s/contents?term_id=%s", infzmBaseURL, id),
		"南方周末"+name+"频道最新文章",
	)
	articles := resp.Data.Contents
	if len(articles) > limit {
		articles = articles[:limit]
	}
	infzmAppendArticles(feed, c, articles)
	return feed, nil
}

func infzmAppendArticles(feed *models.Feed, c *ctxpkg.Context, articles []infzmArticle) {
	for _, a := range articles {
		if a.Subject == "" || a.ID == 0 {
			continue
		}
		link := fmt.Sprintf("%s/contents/%d", infzmBaseURL, a.ID)
		desc := "<p>" + html.EscapeString(a.Introtext) + "</p>"
		if body, err := infzmProfile().Referer(link).
			Fetch(link).GetString(c.Parent(), c.Client()); err == nil {
			if content := infzmExtractContent(body); content != "" {
				desc = content
			}
		}
		item := routeutils.NewItem(a.Subject, link, desc, infzmParseTime(a.PublishTime))
		if item == nil {
			continue
		}
		item.GUID = fmt.Sprintf("infzm-%d", a.ID)
		if a.Author != "" {
			routeutils.SetItemAuthor(item, a.Author, "", "")
		}
		routeutils.AddItem(feed, item)
	}
}

// infzmExtractContent pulls the article body out of an article page.
func infzmExtractContent(pageHTML string) string {
	marker := `class="nfzm-content__content"`
	idx := strings.Index(pageHTML, marker)
	if idx < 0 {
		return ""
	}
	start := strings.Index(pageHTML[idx:], ">")
	if start < 0 {
		return ""
	}
	start += idx + 1
	endTag := `<!--</div>-->`
	end := strings.Index(pageHTML[start:], `<div class="nfzm-content__bottom"`)
	if end < 0 {
		end = strings.Index(pageHTML[start:], "</article>")
	}
	if end < 0 {
		end = len(pageHTML) - start
	}
	content := strings.TrimSpace(pageHTML[start : start+end])
	content = strings.TrimSuffix(content, endTag)
	if len(content) < 40 {
		return ""
	}
	return content
}

func infzmParseTime(s string) time.Time {
	t, err := time.ParseInLocation(infzMTimeLayout, s, infzmLocation)
	if err != nil {
		return time.Time{}
	}
	return t
}
