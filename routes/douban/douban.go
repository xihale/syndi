package routes

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const doubanBaseURL = "https://movie.douban.com"

// doubanWebProfile returns the shared disguise profile for douban pages/APIs.
func doubanWebProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9")
}

var doubanMoviePlayingRoute = routeutils.RouteSpec{
	Path:        "movie/playing",
	Name:        "Now Playing Movies",
	Example:     "douban/movie/playing",
	Maintainers: []string{"xihale"},
	Description: "Movies now playing in Chinese cinemas",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    1 * time.Hour,
	Handler:     DoubanMoviePlayingHandler,
}

var doubanMoviePlayingScoreRoute = routeutils.RouteSpec{
	Path:        "movie/playing/:score",
	Name:        "Now Playing Movies by Score",
	Example:     "douban/movie/playing/8",
	Maintainers: []string{"xihale"},
	Description: "Movies now playing in Chinese cinemas filtered by minimum douban score",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("score", "Minimum douban score filter, e.g. 8"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  DoubanMoviePlayingHandler,
}

var doubanMovieWeeklyRoute = routeutils.RouteSpec{
	Path:        "movie/weekly",
	Name:        "Weekly Best Movies",
	Example:     "douban/movie/weekly",
	Maintainers: []string{"xihale"},
	Description: "Douban weekly best movies ranking (一周口碑榜)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Maximum number of items, default 10, max 30"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  DoubanMovieWeeklyHandler,
}

// DoubanMoviePlayingHandler handles /douban/movie/playing/:score?
func DoubanMoviePlayingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	minScore := 0.0
	if raw := c.Param("score"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			minScore = v
		}
	}
	ctx := c.Parent()

	doc, err := doubanWebProfile().Referer(doubanBaseURL+"/").
		Fetch(doubanBaseURL+"/cinema/nowplaying/beijing/").
		GetHTML(ctx, c.Client())
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		doubanPlayingTitle(minScore),
		doubanBaseURL+"/cinema/nowplaying/",
		"正在上映的电影",
	)
	doc.Each("li.list-item", func(_ int, sel *parser.Selection) {
		score, _ := strconv.ParseFloat(sel.AttrOr("data-score", "0"), 64)
		title := sel.AttrOr("data-title", "")
		subjectID := sel.AttrOr("id", sel.AttrOr("data-subject", ""))
		if title == "" || subjectID == "" || score < minScore {
			return
		}
		desc := fmt.Sprintf("标题：%s<br>评分：%.1f<br>片长：%s<br>制片国家/地区：%s<br>导演：%s<br>主演：%s",
			html.EscapeString(title), score,
			html.EscapeString(sel.AttrOr("data-duration", "")),
			html.EscapeString(sel.AttrOr("data-region", "")),
			html.EscapeString(sel.AttrOr("data-director", "")),
			html.EscapeString(sel.AttrOr("data-actors", "")),
		)
		var sb strings.Builder
		sb.WriteString(desc)
		if poster := sel.Find(".poster img").AttrOr("src", ""); poster != "" {
			sb.WriteString(fmt.Sprintf(`<br><img src="%s">`, html.EscapeString(poster)))
		}
		item := routeutils.NewItem(title, doubanBaseURL+"/subject/"+subjectID, sb.String(), time.Time{})
		if item == nil {
			return
		}
		item.GUID = "douban-movie-" + subjectID
		routeutils.AddItem(feed, item)
	})
	return feed, nil
}

// DoubanMovieWeeklyHandler handles /douban/movie/weekly
func DoubanMovieWeeklyHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 10, 30)
	ctx := c.Parent()

	apiURL := fmt.Sprintf("https://m.douban.com/rexxar/api/v2/subject_collection/movie_weekly_best/items?start=0&count=%d", limit)
	var resp doubanWeeklyResp
	if err := doubanWebProfile().JSONAccept().
		Referer("https://m.douban.com/movie/weekly/").
		Fetch(apiURL).
		GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("豆瓣电影一周口碑榜", "https://m.douban.com/movie/weekly/", "豆瓣一周口碑电影榜")
	for _, entry := range resp.SubjectCollectionItems {
		if entry.Title == "" && entry.URL == "" {
			continue
		}
		link := entry.URL
		if link == "" {
			link = doubanBaseURL + "/subject/" + entry.ID
		}
		rating := "暂无评分"
		if entry.Rating.Value > 0 {
			rating = fmt.Sprintf("%.1f (%d 人评)", entry.Rating.Value, entry.Rating.Count)
		}
		desc := ""
		if poster := entry.Pic.Large; poster != "" {
			desc += fmt.Sprintf(`<img src="%s" alt=""/><br/>`, html.EscapeString(poster))
		}
		desc += html.EscapeString(entry.CardSubtitle) + "<br/>" +
			fmt.Sprintf("排名: %s | 评分: %s<br/>", rankText(entry.Rank), html.EscapeString(rating)) +
			html.EscapeString(entry.Description)

		title := fmt.Sprintf("%s. %s", rankText(entry.Rank), entry.Title)
		item := routeutils.NewItem(title, link, desc, time.Time{})
		if item == nil {
			continue
		}
		if entry.ID != "" {
			item.GUID = "douban-weekly-" + entry.ID
		}
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

func doubanPlayingTitle(minScore float64) string {
	if minScore > 0 {
		return fmt.Sprintf("正在上映的超过 %.1f 分的电影", minScore)
	}
	return "正在上映的电影"
}

func rankText(rank int) string {
	if rank <= 0 {
		return "-"
	}
	return strconv.Itoa(rank)
}

type doubanWeeklyResp struct {
	Start                  int                `json:"start"`
	Count                  int                `json:"count"`
	Total                  int                `json:"total"`
	SubjectCollectionItems []doubanWeeklyItem `json:"subject_collection_items"`
}

type doubanWeeklyItem struct {
	ID           string `json:"id"`
	Rank         int    `json:"rank"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Description  string `json:"description"`
	CardSubtitle string `json:"card_subtitle"`
	Pic          struct {
		Large  string `json:"large"`
		Normal string `json:"normal"`
	} `json:"pic"`
	Rating struct {
		Value float64 `json:"value"`
		Count int     `json:"count"`
	} `json:"rating"`
}
