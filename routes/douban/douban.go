package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const (
	doubanBaseURL    = "https://movie.douban.com"
	doubanWWWBaseURL = "https://www.douban.com"
	doubanMobileURL  = "https://m.douban.com"

	doubanRexxarAPI = "https://m.douban.com/rexxar/api/v2"
)

// doubanWebProfile returns the shared disguise profile for douban pages/APIs.
func doubanWebProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9")
}

// doubanFetchHTML fetches a douban page through the shared disguise profile.
func doubanFetchHTML(ctx context.Context, cl *client.Client, url, referer string) (*parser.Document, error) {
	return doubanWebProfile().Referer(referer).Fetch(url).GetHTML(ctx, cl)
}

// doubanFetchJSON fetches a douban JSON API through an XHR-like disguise
// profile and rejects well-formed error envelopes ({"code":103,...}).
func doubanFetchJSON(ctx context.Context, cl *client.Client, url, referer string, target interface{}) error {
	data, err := doubanWebProfile().JSONAccept().Referer(referer).Fetch(url).GetBytes(ctx, cl)
	if err != nil {
		return err
	}
	return decodeDoubanJSON(url, data, target)
}

// decodeDoubanJSON validates the response envelope and unmarshals the payload.
func decodeDoubanJSON(sourceURL string, data []byte, target interface{}) error {
	var env doubanAPIError
	if err := json.Unmarshal(data, &env); err == nil && env.Code != 0 && env.Code != 200 {
		env.URL = sourceURL
		return &env
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("douban: invalid JSON from %s: %w", sourceURL, err)
	}
	return nil
}

// doubanAPIError models the error envelope douban APIs return on failures
// such as {"code":103,"msg":"need_login"} or blocked apikeys.
type doubanAPIError struct {
	Code             int    `json:"code"`
	Msg              string `json:"msg"`
	LocalizedMessage string `json:"localized_message"`
	URL              string `json:"-"`
}

func (e *doubanAPIError) Error() string {
	msg := e.Msg
	if msg == "" {
		msg = e.LocalizedMessage
	}
	if msg == "" {
		msg = "unknown api error"
	}
	return fmt.Sprintf("douban: api error code %d (%s) from %s", e.Code, msg, e.URL)
}

// doubanSimpleKeyPattern guards path params that are interpolated into URLs.
var doubanSimpleKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// doubanSanitizeKey normalizes an optional path param used inside an upstream
// URL; anything unexpected falls back to fallback.
func doubanSanitizeKey(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	if !doubanSimpleKeyPattern.MatchString(raw) {
		return fallback
	}
	return raw
}

// doubanCollectionKeys returns sorted keys of a category map (stable order
// for docs and ParseEnum).
func doubanCollectionKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// doubanIDFromLink extracts the trailing path segment of a link (e.g. the
// subject/topic id), with query strings and fragments removed.
func doubanIDFromLink(link string) string {
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil || u.Path == "" {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return ""
}

var doubanDateLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

// doubanParseDate parses common douban date strings, zero time on failure.
func doubanParseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range doubanDateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t
		}
	}
	return time.Time{}
}

var doubanPubdateYearPattern = regexp.MustCompile(`(\d{4})(?:-(\d{1,2}))?(?:-(\d{1,2}))?`)

// doubanParsePubdate parses a douban pubdate entry like "2026-08-28(中国大陆)"
// where trailing context in parentheses is common.
func doubanParsePubdate(entries []string) time.Time {
	for _, entry := range entries {
		m := doubanPubdateYearPattern.FindStringSubmatch(entry)
		if m == nil {
			continue
		}
		year, _ := strconv.Atoi(m[1])
		month := 1
		day := 1
		if m[2] != "" {
			month, _ = strconv.Atoi(m[2])
		}
		if m[3] != "" {
			day, _ = strconv.Atoi(m[3])
		}
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	}
	return time.Time{}
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

var doubanMovieWeeklyTypeRoute = routeutils.RouteSpec{
	Path:        "movie/weekly/:type",
	Name:        "Douban Subject Collection Ranking",
	Example:     "douban/movie/weekly/tv_chinese_best_weekly",
	Maintainers: []string{"xihale"},
	Description: "Douban ranking by subject collection id, e.g. movie_weekly_best (一周口碑电影榜), tv_chinese_best_weekly (华语口碑剧集榜)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("type", "Subject collection id from m.douban.com, e.g. movie_weekly_best"),
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

	doc, err := doubanFetchHTML(ctx, c.Client(), doubanBaseURL+"/cinema/nowplaying/beijing/", doubanBaseURL+"/")
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

// DoubanMovieWeeklyHandler handles /douban/movie/weekly/:type?
func DoubanMovieWeeklyHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 10, 30)
	typ := doubanSanitizeKey(c.Param("type"), "movie_weekly_best")
	referer := doubanMobileURL + "/movie/"
	ctx := c.Parent()

	itemsURL := fmt.Sprintf("%s/subject_collection/%s/items?start=0&count=%d", doubanRexxarAPI, typ, limit)
	var coll doubanCollectionResp
	if err := doubanFetchJSON(ctx, c.Client(), itemsURL, referer, &coll); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"豆瓣电影一周口碑榜",
		doubanMobileURL+"/subject_collection/"+typ,
		"豆瓣一周口碑电影榜",
	)
	// Collection metadata is optional decoration; ignore fetch failures.
	metaURL := fmt.Sprintf("%s/subject_collection/%s", doubanRexxarAPI, typ)
	var meta doubanCollectionMeta
	if err := doubanFetchJSON(ctx, c.Client(), metaURL, referer, &meta); err == nil {
		if meta.Title != "" {
			feed.Title = meta.Title
		}
		if meta.Description != "" {
			feed.Description = meta.Description
		}
	}
	doubanAppendCollectionItems(feed, coll.items())
	return feed, nil
}

// doubanCollectionResp is the payload of rexxar subject_collection list APIs.
// Some collections use "subject_collection_items", others ("new_book_*") use
// "items"; both are kept and unified via items().
type doubanCollectionResp struct {
	Start                  int                    `json:"start"`
	Count                  int                    `json:"count"`
	Total                  int                    `json:"total"`
	Items                  []doubanCollectionItem `json:"items"`
	SubjectCollectionItems []doubanCollectionItem `json:"subject_collection_items"`
	Subjects               []doubanCollectionItem `json:"subjects"`
}

// items returns whichever item array is populated.
func (r *doubanCollectionResp) items() []doubanCollectionItem {
	if len(r.SubjectCollectionItems) > 0 {
		return r.SubjectCollectionItems
	}
	if len(r.Items) > 0 {
		return r.Items
	}
	return r.Subjects
}

// doubanCollectionMeta is a rexxar subject_collection info payload.
type doubanCollectionMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Subtitle    string `json:"subtitle"`
}

// doubanRating appears as object or null depending on collection.
type doubanRating struct {
	Value float64 `json:"value"`
	Count int     `json:"count"`
	Max   int     `json:"max"`
}

type doubanPicURL struct {
	URL string `json:"url"`
}

type doubanPicSizes struct {
	Large  string `json:"large"`
	Normal string `json:"normal"`
	Medium string `json:"medium"`
}

type doubanCard struct {
	Content string `json:"content"`
	KindCN  string `json:"kind_cn"`
}

// doubanCollectionItem is the union of fields used across douban rexxar
// collections (rankings, book/music/event lists, frodo coming_soon).
// Only field shapes verified against live payloads are modeled.
type doubanCollectionItem struct {
	ID               string          `json:"id"`
	Rank             int             `json:"rank"`
	Title            string          `json:"title"`
	URL              string          `json:"url"`
	Info             string          `json:"info"`
	Subtype          string          `json:"subtype"`
	PriceRange       string          `json:"price_range"`
	Abstract         string          `json:"abstract"`
	Description      string          `json:"description"`
	CardSubtitle     string          `json:"card_subtitle"`
	NullRatingReason string          `json:"null_rating_reason"`
	RecommendComment string          `json:"recommend_comment"`
	Year             string          `json:"year"`
	Intro            string          `json:"intro"`
	WishCount        float64         `json:"wish_count"`
	Pubdate          []string        `json:"pubdate"`
	Genres           []string        `json:"genres"`
	Directors        []doubanName    `json:"directors"`
	Actors           []doubanName    `json:"actors"`
	Rating           *doubanRating   `json:"rating"`
	Pic              *doubanPicSizes `json:"pic"`
	Cover            *doubanPicURL   `json:"cover"`
	CoverURL         string          `json:"cover_url"`
	Cards            []doubanCard    `json:"cards"`
}

type doubanName struct {
	Name string `json:"name"`
}

// Poster prefers pic.normal > pic.large > cover.url > cover_url.
func (it *doubanCollectionItem) Poster() string {
	if it.Pic != nil {
		if it.Pic.Normal != "" {
			return it.Pic.Normal
		}
		if it.Pic.Large != "" {
			return it.Pic.Large
		}
	}
	if it.Cover != nil && it.Cover.URL != "" {
		return it.Cover.URL
	}
	return it.CoverURL
}

// RatingText renders a human rating line for the item.
func (it *doubanCollectionItem) RatingText() string {
	if it.Rating != nil && it.Rating.Value > 0 {
		count := ""
		if it.Rating.Count > 0 {
			count = fmt.Sprintf("（%d 人评）", it.Rating.Count)
		}
		return fmt.Sprintf("%.1f 分%s", it.Rating.Value, count)
	}
	if it.NullRatingReason != "" {
		return it.NullRatingReason
	}
	return "暂无评分"
}

func doubanLink(it *doubanCollectionItem, fallbackPrefix string) string {
	if it.URL != "" {
		return it.URL
	}
	if it.ID != "" {
		return fallbackPrefix + it.ID + "/"
	}
	return ""
}

func doubanItemTitle(it *doubanCollectionItem) string {
	title := it.Title
	if title != "" && it.Info != "" {
		title = title + "-" + it.Info
	}
	return routeutils.CollapseWhitespace(title)
}

// doubanAppendCollectionItems appends generic card-style items used by
// rankings and latest-list routes.
func doubanAppendCollectionItems(feed *models.Feed, items []doubanCollectionItem) {
	for _, entry := range items {
		link := doubanLink(&entry, "")
		title := doubanItemTitle(&entry)
		if title == "" && link == "" {
			continue
		}
		var sb strings.Builder
		if poster := entry.Poster(); poster != "" {
			sb.WriteString(fmt.Sprintf(`<img src="%s" alt=""/><br/>`, html.EscapeString(poster)))
		}
		if line := routeutils.CollapseWhitespace(entry.CardSubtitle); line != "" {
			sb.WriteString(html.EscapeString(line) + "<br/>")
		} else if line := routeutils.CollapseWhitespace(entry.Info); line != "" {
			sb.WriteString(html.EscapeString(line) + "<br/>")
		}
		sb.WriteString(fmt.Sprintf("评分：%s<br/>", html.EscapeString(entry.RatingText())))
		if text := routeutils.CollapseWhitespace(firstNonEmpty(entry.RecommendComment, entry.Description, entry.Abstract)); text != "" {
			sb.WriteString(html.EscapeString(text))
		}

		item := routeutils.NewItem(title, link, sb.String(), time.Time{})
		if item == nil {
			continue
		}
		if entry.ID != "" {
			item.GUID = "douban-subject-" + entry.ID
		}
		routeutils.AddItem(feed, item)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
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
