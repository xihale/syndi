package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// Legacy douban open API us_box endpoint; one of the long-standing public
// apikeys that still answers this request.
const (
	doubanUSBoxAPI    = "https://api.douban.com/v2/movie/us_box"
	doubanUSBoxAPIKey = "054022eaeae0b00e0fc068c0c0a2102a"
)

var doubanMovieUSBoxRoute = routeutils.RouteSpec{
	Path:        "movie/ustop",
	Name:        "US Box Office Top Movies",
	Example:     "douban/movie/ustop",
	Maintainers: []string{"xihale"},
	Description: "Douban North America box office ranking (北美票房榜)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    6 * time.Hour,
	Handler:     DoubanMovieUSBoxHandler,
}

type doubanUSBoxResp struct {
	Date     string `json:"date"`
	Subjects []struct {
		Box     float64           `json:"box"`
		IsNew   bool              `json:"new"`
		Rank    int               `json:"rank"`
		Subject doubanSubjectInfo `json:"subject"`
	} `json:"subjects"`
}

// doubanSubjectInfo is the nested movie object of the v2 movie API.
type doubanSubjectInfo struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	Alt           string   `json:"alt"`
	Genres        []string `json:"genres"`

	Rating struct {
		Average float64 `json:"average"`
		Stars   string  `json:"stars"`
	} `json:"rating"`

	Images struct {
		Small  string `json:"small"`
		Large  string `json:"large"`
		Medium string `json:"medium"`
	} `json:"images"`
}

// DoubanMovieUSBoxHandler handles /douban/movie/ustop
func DoubanMovieUSBoxHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()

	apiURL := fmt.Sprintf("%s?apikey=%s", doubanUSBoxAPI, doubanUSBoxAPIKey)
	var resp doubanUSBoxResp
	if err := doubanFetchJSON(ctx, c.Client(), apiURL, doubanBaseURL+"/chart", &resp); err != nil {
		return nil, err
	}
	if len(resp.Subjects) == 0 {
		return nil, fmt.Errorf("douban: us_box returned no data")
	}

	description := "北美票房榜（周票房）"
	if resp.Date != "" {
		description = fmt.Sprintf("北美票房榜，统计周期：%s", resp.Date)
	}
	feed := routeutils.NewFeed("豆瓣电影北美票房榜", doubanBaseURL+"/chart", description)
	for _, entry := range resp.Subjects {
		if item := buildDoubanUSBoxItem(entry.Rank, entry.Box, entry.Subject); item != nil {
			routeutils.AddItem(feed, item)
		}
	}
	return feed, nil
}

func buildDoubanUSBoxItem(rank int, box float64, subject doubanSubjectInfo) *models.Item {
	title := routeutils.CollapseWhitespace(subject.Title)
	link := subject.Alt
	if title == "" || link == "" {
		return nil
	}

	var sb strings.Builder
	if subject.Images.Large != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s" alt=""/><br>`, html.EscapeString(subject.Images.Large)))
	}
	sb.WriteString(fmt.Sprintf("标题：%s<br>", html.EscapeString(title)))
	if len(subject.Genres) > 0 {
		sb.WriteString(fmt.Sprintf("影片类型：%s<br>", html.EscapeString(strings.Join(subject.Genres, " | "))))
	}
	sb.WriteString(fmt.Sprintf("评分：%s<br>", html.EscapeString(doubanStarText(subject.Rating.Average, subject.Rating.Stars))))
	if rank > 0 {
		sb.WriteString(fmt.Sprintf("排名：第 %s 名<br>", rankText(rank)))
	}
	if box > 0 {
		sb.WriteString(fmt.Sprintf("周末票房：%s", html.EscapeString(doubanFormatBox(box))))
	}

	item := routeutils.NewItem(title, link, sb.String(), time.Time{})
	if item == nil {
		return nil
	}
	item.GUID = "douban-ustop-" + firstNonEmpty(subject.ID, doubanIDFromLink(link))
	return item
}

// doubanStarText renders the rating like upstream ("无" when unstarred).
func doubanStarText(average float64, stars string) string {
	if stars == "00" || average <= 0 {
		return "无"
	}
	return fmt.Sprintf("%.1f", average)
}

// doubanFormatBox renders a CNY box office value with 亿/万 units.
func doubanFormatBox(v float64) string {
	switch {
	case v >= 1e8:
		return fmt.Sprintf("%.2f亿", v/1e8)
	case v >= 1e4:
		return fmt.Sprintf("%.0f万", v/1e4)
	default:
		return fmt.Sprintf("%.0f元", v)
	}
}
