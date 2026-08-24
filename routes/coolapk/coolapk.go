package routes

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const coolapkAPIBase = "https://api.coolapk.com/v6/page/dataList?url="

// coolapkAppToken replicates the CoolMarket app token algorithm used by the
// upstream client: md5(base64(tokenSource)) + deviceID + hex(now).
func coolapkAppToken() string {
	deviceID := coolapkRandomUUID()
	now := time.Now().Unix()
	hexNow := "0x" + strconv.FormatInt(now, 16)
	md5Now := fmt.Sprintf("%x", md5.Sum([]byte(strconv.FormatInt(now, 10))))
	source := "token://com.coolapk.market/c67ef5943784d09750dcfbb31020f0ab?" + md5Now + "$" + deviceID + "&com.coolapk.market"
	b64 := base64.StdEncoding.EncodeToString([]byte(source))
	tokenHash := fmt.Sprintf("%x", md5.Sum([]byte(b64)))
	return tokenHash + deviceID + hexNow
}

// coolapkRandomUUID generates a random RFC 4122 version 4 UUID string.
func coolapkRandomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

func coolapkHeaders() map[string]string {
	return map[string]string{
		"X-Requested-With": "XMLHttpRequest",
		"X-App-Id":         "com.coolapk.market",
		"X-App-Token":      coolapkAppToken(),
		"X-Sdk-Int":        "29",
		"X-Sdk-Locale":     "zh-CN",
		"X-App-Version":    "11.0",
		"X-Api-Version":    "11",
		"X-App-Code":       "2101202",
		"User-Agent":       "Dalvik/2.1.0 (Linux; U; Android 10; Redmi K30 5G MIUI/V12.0.3.0.QGICMXM) (#Build; Redmi; Redmi K30 5G; QKQ1.191222.002 test-keys; 10) +CoolMarket/11.0-2101202",
	}
}

var coolapkHotRoute = routeutils.RouteSpec{
	Path:        "hot",
	Name:        "Hot List",
	Example:     "coolapk/hot",
	Maintainers: []string{"xihale"},
	Description: "Coolapk hot ranking list (today hot)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters:  nil,
	CacheTTL:    30 * time.Minute,
	Handler:     CoolapkHotHandler,
}

var coolapkHotTypeRoute = routeutils.RouteSpec{
	Path:        "hot/:type",
	Name:        "Hot List by Type",
	Example:     "coolapk/hot/jrrm",
	Maintainers: []string{"xihale"},
	Description: "Coolapk hot ranking list: jrrm today hot, dzb likes, scb favorites, plb replies",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("type", "List type: jrrm today hot, dzb likes, scb favorites, plb replies"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  CoolapkHotHandler,
}

var coolapkToutiaoRoute = routeutils.RouteSpec{
	Path:        "toutiao",
	Name:        "Headlines",
	Example:     "coolapk/toutiao",
	Maintainers: []string{"xihale"},
	Description: "Coolapk headline feed (latest posts)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters:  nil,
	CacheTTL:    30 * time.Minute,
	Handler:     CoolapkToutiaoHandler,
}

var coolapkToutiaoTypeRoute = routeutils.RouteSpec{
	Path:        "toutiao/:type",
	Name:        "Headlines by Type",
	Example:     "coolapk/toutiao/digest",
	Maintainers: []string{"xihale"},
	Description: "Coolapk headline feeds: headline or digest latest posts",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("type", "Feed type: headline or digest"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  CoolapkToutiaoHandler,
}

// CoolapkHotHandler handles /coolapk/hot/:type?
func CoolapkHotHandler(c *ctxpkg.Context) (*models.Feed, error) {
	typeName := c.Param("type")
	var fragment, title string
	switch typeName {
	case "", "jrrm":
		title = "今日热门"
		fragment = "/feed/statList?cacheExpires=300&statType=day&sortField=detailnum&title=今日热门"
	case "dzb":
		title = "点赞榜"
		fragment = "/feed/statList?statType=day&sortField=likenum&title=日榜"
	case "scb":
		title = "收藏榜"
		fragment = "/feed/statList?statType=day&sortField=favnum&title=日榜"
	case "plb":
		title = "评论榜"
		fragment = "/feed/statList?statType=day&sortField=replynum&title=日榜"
	default:
		return nil, fmt.Errorf("unknown coolapk hot type %q, allowed: jrrm, dzb, scb, plb", typeName)
	}
	return fetchCoolapkDataList(c, fragment, title)
}

// CoolapkToutiaoHandler handles /coolapk/toutiao/:type?
func CoolapkToutiaoHandler(c *ctxpkg.Context) (*models.Feed, error) {
	typeName := c.Param("type")
	var fragment, title string
	switch typeName {
	case "", "headline":
		title = "历史头条"
		fragment = "/feed/headlineV8List?type=0,5,9,8,12,10,11,13&title=历史头条"
	case "digest":
		title = "最新动态"
		fragment = "/feed/digestList?type=0,5,12,10,11,13,8,9&title=最新动态"
	default:
		return nil, fmt.Errorf("unknown coolapk toutiao type %q, allowed: headline, digest", typeName)
	}
	return fetchCoolapkDataList(c, fragment, title)
}

// fetchCoolapkDataList requests a dataList endpoint and maps feed entries.
func fetchCoolapkDataList(c *ctxpkg.Context, fragment, title string) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)
	ctx := c.Parent()

	apiURL := coolapkAPIBase + url.QueryEscape(fragment) + "&page=1"
	var resp coolapkDataListResp
	if err := routeutils.GetJSONWithHeaders(ctx, c.Client(), apiURL, coolapkHeaders(), &resp); err != nil {
		fmt.Println("COOLAPK_DEBUG fetch error:", err)
		return nil, err
	}
	fmt.Println("COOLAPK_DEBUG entries:", len(resp.Flattened()), "datalen:", len(resp.Data))
	n := len(resp.Data)
	if n > 220 {
		n = 220
	}
	fmt.Println("COOLAPK_DEBUG head:", string(resp.Data[:n]))

	feed := routeutils.NewFeed("酷安 "+title, "https://www.coolapk.com/", "热榜-"+title)
	for _, entry := range resp.Flattened() {
		if limit > 0 && len(feed.Items) >= limit {
			break
		}
		item := buildCoolapkItem(entry)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

// buildCoolapkItem converts one feed entity into a feed item.
func buildCoolapkItem(entry coolapkEntry) *models.Item {
	if entry.URL == "" || entry.Message == "" && entry.Title == "" {
		return nil
	}
	link := "https://www.coolapk.com" + entry.URL

	message := entry.Message
	if message == "" {
		message = entry.Title
	}
	desc := "<p>" + strings.ReplaceAll(html.EscapeString(message), "\n", "<br/>") + "</p>"
	for _, pic := range entry.PicArr {
		if pic != "" {
			desc += fmt.Sprintf(`<img src="%s"/>`, html.EscapeString(pic))
		}
	}

	title := coolapkTitleFromMessage(message)
	if entry.Type == 10 || entry.Type == 11 {
		if entry.MessageTitle != "" {
			title = entry.MessageTitle + " 更多:" + title
		}
	}

	var pubDate time.Time
	if entry.Dateline > 0 {
		pubDate = time.Unix(int64(entry.Dateline), 0)
	}
	item := routeutils.NewItem(title, link, desc, pubDate)
	if item == nil {
		return nil
	}
	if entry.ID != "" {
		item.GUID = "coolapk-feed-" + entry.ID.String()
	}
	if entry.Username != "" {
		routeutils.SetAuthor(item, entry.Username)
	}
	return item
}

// coolapkTitleFromMessage extracts a plain-text one-line title from a message.
func coolapkTitleFromMessage(message string) string {
	firstLine := message
	if idx := strings.IndexByte(firstLine, '\n'); idx >= 0 {
		firstLine = firstLine[:idx]
	}
	// Messages may embed inline HTML (topic tags etc.); keep titles plain.
	if strings.Contains(firstLine, "<") {
		firstLine = coolapkTagStripRe.ReplaceAllString(firstLine, "")
	}
	firstLine = html.UnescapeString(strings.TrimSpace(firstLine))
	runes := []rune(firstLine)
	if len(runes) > 80 {
		return string(runes[:77]) + "..."
	}
	if firstLine == "" {
		return "(无标题内容)"
	}
	return firstLine
}

// coolapkTagStripRe removes inline HTML markup for plain-text titles.
var coolapkTagStripRe = regexp.MustCompile(`<[^>]*>`)

type coolapkDataListResp struct {
	Data json.RawMessage `json:"data"`
}

// Flattened decodes data as either a plain array of entities or card wrappers.
// Entries are decoded individually so one odd-shaped row cannot void the
// whole list (mixed lists like digestList do occur in the wild).
func (r *coolapkDataListResp) Flattened() []coolapkEntry {
	var raws []json.RawMessage
	if err := json.Unmarshal(r.Data, &raws); err != nil {
		return nil
	}
	var entries []coolapkEntry
	for _, raw := range raws {
		var e coolapkEntry
		if err := json.Unmarshal(raw, &e); err != nil {
			continue
		}
		if e.EntityType == "card" {
			entries = append(entries, e.Entities...)
		} else {
			entries = append(entries, e)
		}
	}
	return entries
}

type coolapkEntry struct {
	EntityType   string         `json:"entityType"`
	ID           json.Number    `json:"id"`
	Type         coolapkInt     `json:"type"`
	Title        string         `json:"title"`
	MessageTitle string         `json:"message_title"`
	Message      string         `json:"message"`
	Dateline     coolapkTime    `json:"dateline"`
	URL          string         `json:"url"`
	Username     string         `json:"username"`
	PicArr       flexStrings    `json:"picArr"`
	Entities     []coolapkEntry `json:"entities"`
}

// flexStrings tolerates picArr appearing as an array, a bare string or null.
type flexStrings []string

func (s *flexStrings) UnmarshalJSON(b []byte) error {
	var arr []string
	if err := json.Unmarshal(b, &arr); err == nil {
		*s = arr
		return nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		if str != "" {
			*s = []string{str}
		}
		return nil
	}
	*s = nil
	return nil
}

// coolapkInt accepts JSON numbers as numbers or strings; coolapk serves some
// fields either way depending on the endpoint.
type coolapkInt int64

func (t *coolapkInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*t = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		*t = 0
		return nil // tolerate unexpected shapes instead of dropping the entry
	}
	*t = coolapkInt(v)
	return nil
}

// coolapkTime accepts unix timestamps as numbers or strings.
type coolapkTime = coolapkInt
