// Package routes ports the RSSHub odaily (Odaily 星球日报) namespace.
//
// Upstream note: RSSHub's TypeScript implementation targets the legacy
// www.odaily.news/api/pp/... app-front endpoints, which are dead as of
// 2026-08 (they answer with the Next.js 404 page). The site now runs on a
// Next.js frontend backed by https://web-api.odaily.news, discovered from the
// deployed JS bundles and verified against live responses:
//
//	GET /post/page?page=1&size=N                       latest posts
//	GET /post/detail/{id}                              full article HTML + author
//	GET /newsflash/page?page=1&size=N                  newsflashes
//	GET /hotRank/list?hotRankType=...&entityType=POST  daily/weekly hot ranking
package routes

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/disguise"
)

const (
	odailyRootURL = "https://www.odaily.news"
	odailyAPIBase = "https://web-api.odaily.news"
)

// odailyInt64 accepts both JSON numbers and quoted strings; the API has been
// observed switching id/timestamp encodings between deployments.
type odailyInt64 int64

func (n *odailyInt64) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*n = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*n = odailyInt64(v)
	return nil
}

func (n odailyInt64) Int64() int64 { return int64(n) }

func (n odailyInt64) String() string { return strconv.FormatInt(int64(n), 10) }

// odailyProfile builds an XHR-like request profile for web-api.odaily.news.
func odailyProfile() *disguise.Profile {
	return disguise.Chrome().
		JSONAccept().
		Lang("zh-CN,zh;q=0.9,en;q=0.8").
		Referer(odailyRootURL + "/")
}

// odailyPageResp is the shared envelope of paginated list endpoints.
type odailyPageResp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Data    struct {
		List []odailyFeedItem `json:"list"`
	} `json:"data"`
}

// odailyFeedItem is one entry of post/page and newsflash/page lists. The two
// endpoints share a shape: posts fill Summary/Cover/Tags while newsflashes
// fill Description/NewsURL.
type odailyFeedItem struct {
	ID               odailyInt64 `json:"id"`
	Title            string      `json:"title"`
	Cover            string      `json:"cover"`
	Summary          string      `json:"summary"`
	Description      string      `json:"description"`
	PublishTimestamp odailyInt64 `json:"publishTimestamp"`
	NewsURL          string      `json:"newsUrl"`
	Tags             []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

// odailyDetail is the payload of /post/detail/{id}.
type odailyDetail struct {
	ID               odailyInt64 `json:"id"`
	Title            string      `json:"title"`
	Cover            string      `json:"cover"`
	Summary          string      `json:"summary"`
	Content          string      `json:"content"`
	PublishTimestamp odailyInt64 `json:"publishTimestamp"`
	Author           struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Title    string `json:"title"`
	} `json:"author"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

type odailyDetailResp struct {
	Code    int           `json:"code"`
	Success bool          `json:"success"`
	Data    *odailyDetail `json:"data"`
}

// fetchOdailyDetails retrieves /post/detail/{id} for each id with a bounded
// worker pool. Failures degrade gracefully: missing ids simply stay absent
// from the returned map.
func fetchOdailyDetails(ctx context.Context, cl *client.Client, ids []string) map[string]*odailyDetail {
	const workers = 5
	out := make(map[string]*odailyDetail, len(ids))
	if len(ids) == 0 {
		return out
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var resp odailyDetailResp
			url := fmt.Sprintf("%s/post/detail/%s", odailyAPIBase, id)
			if err := odailyProfile().Fetch(url).GetJSON(ctx, cl, &resp); err != nil || resp.Data == nil {
				return
			}
			mu.Lock()
			out[id] = resp.Data
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return out
}

// odailyTime converts an epoch-milliseconds field into time.Time.
func odailyTime(ms odailyInt64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms.Int64())
}

// odailyCleanImage strips OSS resize suffixes (?x-oss-process=..., !heading)
// so feeds embed original images, mirroring upstream's img cleanup.
func odailyCleanImage(raw string) string {
	if i := strings.Index(raw, "?"); i >= 0 {
		raw = raw[:i]
	}
	return strings.ReplaceAll(raw, "!heading", "")
}
