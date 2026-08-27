package routes

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// Douban frodo app API credentials used for the signed TV requests, matching
// the upstream implementation.
const (
	doubanFrodoTVComingURL = "https://frodo.douban.com/api/v2/tv/coming_soon"
	doubanFrodoAPIKey      = "0dad551ec0f84ed02907ff5c42e8ec70"
	doubanFrodoAPISecret   = "bf7dddc7c9cfe6f7"
	doubanFrodoUserAgent   = "api-client/1 com.douban.frodo/7.22.0.beta9(231) Android/23 product/Mate 40 vendor/HUAWEI model/Mate 40 brand/HUAWEI rom/android network/wifi platform/AndroidPad"
)

var doubanTVComingRoute = routeutils.RouteSpec{
	Path:        "tv/coming",
	Name:        "Coming Soon TV Series",
	Example:     "douban/tv/coming",
	Maintainers: []string{"xihale"},
	Description: "Douban TV series coming soon (即将播出的剧集)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    6 * time.Hour,
	Handler:     DoubanTVComingHandler,
}

var doubanTVComingSortRoute = routeutils.RouteSpec{
	Path:        "tv/coming/:sortBy",
	Name:        "Coming Soon TV Series by Sort",
	Example:     "douban/tv/coming/time",
	Maintainers: []string{"xihale"},
	Description: "Douban TV series coming soon (即将播出的剧集)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("sortBy", "Sort order: hot (default) or time"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  DoubanTVComingHandler,
}

// doubanFrodoSign computes the HMAC-SHA1 request signature the frodo API
// expects (_sig).
func doubanFrodoSign(path, ts string) string {
	mac := hmac.New(sha1.New, []byte(doubanFrodoAPISecret))
	message := fmt.Sprintf("GET&%s&%s", url.QueryEscape(path), ts)
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// fetchDoubanFrodoJSON uses the frodo app client User-Agent expected by the
// signed API and validates the response envelope.
func fetchDoubanFrodoJSON(ctx context.Context, cl *client.Client, rawURL string, target interface{}) error {
	data, err := disguise.Custom(doubanFrodoUserAgent).
		Accept("application/json").
		Referer("https://frodo.douban.com/").
		Fetch(rawURL).
		GetBytes(ctx, cl)
	if err != nil {
		return err
	}
	return decodeDoubanJSON(rawURL, data, target)
}

// DoubanTVComingHandler handles /douban/tv/coming/:sortBy?
//
// The server always pulls hot-sorted data; sortBy re-orders locally like
// upstream.
func DoubanTVComingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	sortBy := routeutils.ParseEnum(c.Param("sortBy"), "hot", "hot", "time")
	ctx := c.Parent()
	cl := c.Client()

	ts := time.Now().Format("20060102")
	apiURL := fmt.Sprintf("%s?start=0&count=30&sortby=hot&os_rom=android&apiKey=%s&_ts=%s&_sig=%s",
		doubanFrodoTVComingURL,
		url.QueryEscape(doubanFrodoAPIKey),
		ts,
		url.QueryEscape(doubanFrodoSign("/api/v2/tv/coming_soon", ts)),
	)

	var resp doubanCollectionResp
	if err := fetchDoubanFrodoJSON(ctx, cl, apiURL, &resp); err != nil {
		return nil, err
	}
	subjects := resp.items()
	if len(subjects) == 0 {
		return nil, fmt.Errorf("douban tv/coming: no data returned")
	}

	if sortBy == "time" {
		sort.SliceStable(subjects, func(i, j int) bool {
			ti := doubanParsePubdate(subjects[i].Pubdate)
			tj := doubanParsePubdate(subjects[j].Pubdate)
			if !ti.Equal(tj) {
				return ti.Before(tj)
			}
			return subjects[i].WishCount > subjects[j].WishCount
		})
	} else {
		sort.SliceStable(subjects, func(i, j int) bool {
			return subjects[i].WishCount > subjects[j].WishCount
		})
	}

	feed := routeutils.NewFeed(
		"豆瓣剧集-即将播出",
		doubanBaseURL+"/tv/",
		fmt.Sprintf("即将播出的剧集，排序：%s", sortBy),
	)
	for _, subject := range subjects {
		if item := buildDoubanTVComingItem(subject); item != nil {
			routeutils.AddItem(feed, item)
		}
	}
	return feed, nil
}

func buildDoubanTVComingItem(subject doubanCollectionItem) *models.Item {
	title := routeutils.CollapseWhitespace(subject.Title)
	link := firstNonEmpty(subject.URL, doubanBaseURL+"/subject/"+subject.ID+"/")
	if title == "" || link == "" {
		return nil
	}

	description := html.EscapeString(routeutils.CollapseWhitespace(subject.Intro))
	if subject.WishCount > 0 {
		wish := fmt.Sprintf("想看人数：%.0f", subject.WishCount)
		description = strings.TrimSpace(wish + "，" + description)
	}

	item := routeutils.NewItem(title, link, description, doubanParsePubdate(subject.Pubdate))
	if item == nil {
		return nil
	}
	item.GUID = "douban-tv-coming-" + firstNonEmpty(subject.ID, doubanIDFromLink(link))
	routeutils.SetCategories(item, subject.Genres...)
	return item
}
