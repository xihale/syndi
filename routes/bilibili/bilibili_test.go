package routes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/xihale/syndi/internal/routeutils"
	"github.com/xihale/syndi/pkg/models"
)

// loadFixture reads a testdata JSON file; all parsing tests are offline.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func decodeFixture(t *testing.T, name string, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(loadFixture(t, name), target); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
}

func requireNonEmptyFeed(t *testing.T, feed *models.Feed, want int) {
	t.Helper()
	if feed == nil {
		t.Fatal("expected feed")
	}
	if len(feed.Items) < want {
		t.Fatalf("expected at least %d items, got %d", want, len(feed.Items))
	}
	for _, item := range feed.Items {
		if item.Title == "" || item.Link == "" || item.GUID == "" {
			t.Fatalf("item missing title/link/guid: %+v", item)
		}
	}
}

// --- partion ---

func TestPartionFeedFromNewlistFixture(t *testing.T) {
	var resp bilibiliNewListResp
	decodeFixture(t, "newlist.json", &resp)
	if err := resp.Err(); err != nil {
		t.Fatal(err)
	}
	name := bilibiliZoneName(resp.Data.Archives, "4")
	feed := routeutils.NewFeed("test", "https://www.bilibili.com", "test")
	appendPartionVideos(feed, resp.Data.Archives, 30, true)

	requireNonEmptyFeed(t, feed, 3)
	first := feed.Items[0]
	if !strings.HasPrefix(first.GUID, "bilibili-partion-") {
		t.Fatalf("unexpected guid %q", first.GUID)
	}
	if first.PubDate.IsZero() {
		t.Fatal("expected pubdate from newlist")
	}
	if name == "" || strings.Contains(name, "未知") {
		t.Fatalf("zone name should derive from tname, got %q", name)
	}
}

// --- partion ranking ---

func TestPartionRankingFromCateSearchFixture(t *testing.T) {
	var resp bilibiliCateSearchResp
	decodeFixture(t, "cate_search.json", &resp)

	feed := routeutils.NewFeed("test", "https://www.bilibili.com", "test")
	routeutils.AppendMappedItems(feed, resp.Result, 0, mapCateVideo)

	requireNonEmptyFeed(t, feed, 3)
	first := feed.Items[0]
	if !strings.HasPrefix(first.GUID, "bilibili-partion-ranking-") {
		t.Fatalf("unexpected guid %q", first.GUID)
	}
	if first.PubDate.IsZero() {
		t.Fatal("expected pubdate parsed from cate search pubdate string")
	}
	if first.Author == nil || first.Author.Name == "" {
		t.Fatal("expected author on hot rank item")
	}
}

// --- video/page ---

func TestVideoPageFeedFromViewFixture(t *testing.T) {
	var resp bilibiliViewResp
	decodeFixture(t, "view.json", &resp)

	feed := bilibiliVideoPageFeed(&resp, true, 100)
	requireNonEmptyFeed(t, feed, 6)
	if !strings.HasPrefix(feed.Items[0].GUID, "bilibili-video-page-") {
		t.Fatalf("unexpected guid %q", feed.Items[0].GUID)
	}
	if !strings.Contains(feed.Items[0].Link, "?p=6") {
		t.Fatalf("expected newest-first episode link, got %q", feed.Items[0].Link)
	}
	if !strings.Contains(feed.Items[0].Description, "<iframe") {
		t.Fatal("expected embedded player when embed enabled")
	}

	noEmbed := bilibiliVideoPageFeed(&resp, false, 100)
	if strings.Contains(noEmbed.Items[0].Description, "<iframe") {
		t.Fatal("embed disabled must not emit iframe")
	}
}

// --- video/reply ---

func TestReplyFeedFromReplyMainFixture(t *testing.T) {
	var resp bilibiliReplyResp
	decodeFixture(t, "reply_main.json", &resp)

	feed := bilibiliReplyFeed("任天堂明星大乱斗把（bei）群友吊起来打 fit.Cph、小金金", "https://www.bilibili.com/video/BV1i7411M7N9", 91257161, &resp)
	requireNonEmptyFeed(t, feed, 2)
	first := feed.Items[0]
	if !strings.HasPrefix(first.GUID, "bilibili-video-reply-") {
		t.Fatalf("unexpected guid %q", first.GUID)
	}
	if first.Author == nil || first.Author.Name == "" {
		t.Fatal("expected comment author")
	}
}

// --- bangumi/media ---

func TestBangumiFeedFromPgcFixtures(t *testing.T) {
	var media bilibiliPgcMediaResp
	decodeFixture(t, "pgc_media.json", &media)
	var section bilibiliPgcSectionResp
	decodeFixture(t, "pgc_section.json", &section)

	feed := bilibiliBangumiFeed(&media, &section, true)
	requireNonEmptyFeed(t, feed, 1)
	if feed.Title != media.Result.Title {
		t.Fatalf("expected season title %q, got %q", media.Result.Title, feed.Title)
	}
	wantEpisodes := len(section.Result.MainSection.Episodes)
	for _, sec := range section.Result.Section {
		wantEpisodes += len(sec.Episodes)
	}
	if len(feed.Items) != wantEpisodes {
		t.Fatalf("expected %d episodes, got %d", wantEpisodes, len(feed.Items))
	}
	if !strings.Contains(feed.Items[0].Description, "seasonId=21680") {
		t.Fatal("expected OGV player iframe for episodes")
	}
	if !strings.HasPrefix(feed.Items[0].GUID, "bilibili-bangumi-21680-") {
		t.Fatalf("unexpected guid %q", feed.Items[0].GUID)
	}
}

// --- audio ---

func TestAudioFeedFromMenuFixtures(t *testing.T) {
	var menu bilibiliAudioMenuResp
	decodeFixture(t, "audio_menu.json", &menu)
	var songs bilibiliAudioSongsResp
	decodeFixture(t, "audio_songs.json", &songs)

	feed := bilibiliAudioFeed(&menu, &songs)
	requireNonEmptyFeed(t, feed, 3)
	if feed.Title != menu.Data.Title {
		t.Fatalf("expected menu title, got %q", feed.Title)
	}
	if !strings.HasPrefix(feed.Items[0].GUID, "bilibili-audio-") {
		t.Fatalf("unexpected guid %q", feed.Items[0].GUID)
	}
	if feed.Items[0].Author == nil || feed.Items[0].Author.Name == "" {
		t.Fatal("expected song author")
	}
}

// --- readlist ---

func TestReadlistFeedFromArticlesFixture(t *testing.T) {
	var resp bilibiliReadlistResp
	decodeFixture(t, "readlist.json", &resp)

	feed := bilibiliReadlistFeed("25611", "https://www.bilibili.com/read/readlist/rl25611", &resp)
	requireNonEmptyFeed(t, feed, 3)
	if !strings.HasPrefix(feed.Items[0].GUID, "bilibili-readlist-25611-") {
		t.Fatalf("unexpected guid %q", feed.Items[0].GUID)
	}
	if feed.Author == nil || feed.Author.Name != resp.Data.Author.Name {
		t.Fatal("expected feed author to be the collection author")
	}
}

// --- live room ---

func TestLiveRoomFeedFromGetInfoFixture(t *testing.T) {
	var info bilibiliLiveInfoResp
	decodeFixture(t, "live_room_info_offline.json", &info)
	info.Data.LiveStatus = 1

	feed := bilibiliLiveRoomFeed("哔哩哔哩音悦台", "哔哩哔哩音悦台", "23058", &info)
	requireNonEmptyFeed(t, feed, 1)
	item := feed.Items[0]
	if !strings.HasPrefix(item.GUID, "bilibili-live-room-") {
		t.Fatalf("unexpected guid %q", item.GUID)
	}
	if item.PubDate.IsZero() {
		t.Fatal("expected parsed live_time pubdate")
	}

	info.Data.LiveStatus = 0
	offline := bilibiliLiveRoomFeed("哔哩哔哩音悦台", "哔哩哔哩音悦台", "23058", &info)
	if len(offline.Items) != 0 {
		t.Fatalf("offline room should yield an empty feed, got %d items", len(offline.Items))
	}
}

// --- live search ---

func TestLiveSearchFeedFromSearchFixture(t *testing.T) {
	var resp bilibiliLiveSearchResp
	decodeFixture(t, "livesearch.json", &resp)

	feed := bilibiliLiveSearchFeed("解密", &resp)
	requireNonEmptyFeed(t, feed, 2)
	if !strings.HasPrefix(feed.Items[0].GUID, "bilibili-live-search-") {
		t.Fatalf("unexpected guid %q", feed.Items[0].GUID)
	}
	if strings.Contains(feed.Items[0].Title, "<em") {
		t.Fatal("highlight tags must be stripped from titles")
	}
}

// --- vsearch ---

func TestVsearchFeedFromSearchTypeFixture(t *testing.T) {
	var resp bilibiliSearchTypeResp
	decodeFixture(t, "vsearch.json", &resp)

	link := "https://search.bilibili.com/all?keyword=rsshub&order=pubdate"
	feed := bilibiliVsearchFeed("rsshub", "pubdate", true, link, &resp)
	requireNonEmptyFeed(t, feed, 2)
	first := feed.Items[0]
	if !strings.HasPrefix(first.GUID, "bilibili-vsearch-") {
		t.Fatalf("unexpected guid %q", first.GUID)
	}
	if first.PubDate.IsZero() {
		t.Fatal("expected pubdate")
	}
	if first.Author == nil || first.Author.Name != resp.Data.Result[0].Author {
		t.Fatal("expected up author on search results")
	}
	if !strings.Contains(first.Description, "Play: 665") {
		t.Fatal("expected stats metadata line in description")
	}
}

// --- weekly ---

func TestWeeklyFeedFromSelectedFixture(t *testing.T) {
	var selected bilibiliWeeklySelectedResp
	decodeFixture(t, "weekly_selected.json", &selected)
	current := struct {
		Number int64  `json:"number"`
		Name   string `json:"name"`
		Status int64  `json:"status"`
	}{Number: 387, Name: "2026第387期 08.14 - 08.20"}

	feed := bilibiliWeeklyFeed(&current, &selected, true)
	requireNonEmptyFeed(t, feed, 2)
	if !strings.HasPrefix(feed.Items[0].GUID, "bilibili-weekly-BV") {
		t.Fatalf("unexpected guid %q", feed.Items[0].GUID)
	}
	if !strings.Contains(feed.Items[0].Description, current.Name) {
		t.Fatal("expected week name in description")
	}
}

// --- user/video (wbi arc search) ---

func TestUserVideoFeedFromArcSearchFixture(t *testing.T) {
	var resp bilibiliArcSearchResp
	decodeFixture(t, "arc_search.json", &resp)

	feed := routeutils.NewFeed("test feed", "https://space.bilibili.com/946974", "test")
	appendUserVideos(feed, resp.Data.List.Vlist, true, "946974")
	requireNonEmptyFeed(t, feed, 2)

	first := feed.Items[0]
	if !strings.HasPrefix(first.GUID, "bilibili-user-video-") {
		t.Fatalf("unexpected guid %q", first.GUID)
	}
	if first.Link != "https://www.bilibili.com/video/BV11x411c7ym" {
		t.Fatalf("unexpected link %q", first.Link)
	}
	if feed.Items[1].Link != "https://www.bilibili.com/video/av9469740" {
		t.Fatalf("aid fallback link expected, got %q", feed.Items[1].Link)
	}
}

// --- user/dynamic (polymer feed/space) ---

func TestUserDynamicFeedFromPolymerFixture(t *testing.T) {
	var resp bilibiliDynamicFeedResp
	decodeFixture(t, "dynamic_feed.json", &resp)

	feed := routeutils.NewFeed("test feed", "https://space.bilibili.com/946974/dynamic", "test")
	appendUserDynamics(feed, resp.Data.Items, parseDynamicParams(""))
	requireNonEmptyFeed(t, feed, 3)

	videoItem := feed.Items[0]
	if videoItem.Link != "https://www.bilibili.com/video/BV1xx411c7XX" {
		t.Fatalf("archive dynamic should link to video, got %q", videoItem.Link)
	}
	if !strings.HasPrefix(videoItem.GUID, "bilibili-user-dynamic-") {
		t.Fatalf("unexpected guid %q", videoItem.GUID)
	}

	opusItem := feed.Items[1]
	if opusItem.Link != "https://t.bilibili.com/100000000000000002" {
		t.Fatalf("opus dynamic should link back to t.bilibili.com, got %q", opusItem.Link)
	}
	if strings.Count(opusItem.Description, "<img") != 2 {
		t.Fatal("opus images should be rendered")
	}

	liveItem := feed.Items[2]
	if liveItem.Link != "https://live.bilibili.com/23058" {
		t.Fatalf("live rcmd dynamic should link to room, got %q", liveItem.Link)
	}

	opts := parseDynamicParams("embed=0&showEmoji=1")
	if opts.Embed {
		t.Fatal("embed=0 route param must disable embedding")
	}
}

// --- helpers ---

func TestParseDynamicParams(t *testing.T) {
	cases := []struct {
		raw   string
		embed bool
	}{
		{"", true},
		{"showEmoji=1", true},
		{"embed=0", false},
		{"embed=1", true},
		{"showEmoji=1&embed=0&useAvid=1", false},
	}
	for _, c := range cases {
		if got := parseDynamicParams(c.raw); got.Embed != c.embed {
			t.Fatalf("parseDynamicParams(%q).Embed = %v, want %v", c.raw, got.Embed, c.embed)
		}
	}
}

func TestBilibiliSignWbi(t *testing.T) {
	imgKey := "7cd084941338f37aa62adbb7ded5dd16"
	subKey := "4932caff21ff8b5f52ce9619be6abfa2"
	params := make(map[string][]string, 2)
	params["mid"] = []string{"946974"}
	params["ps"] = []string{"10"}

	now := int64(1700000000)
	signed := bilibiliSignWbi(params, imgKey, subKey, now)
	wts := signed.Get("wts")
	if wts != "1700000000" {
		t.Fatalf("expected wts parameter, got %q", wts)
	}
	if len(signed.Get("w_rid")) != 32 {
		t.Fatalf("expected md5 hex w_rid, got %q", signed.Get("w_rid"))
	}
	// deterministic given same inputs
	again := bilibiliSignWbi(params, imgKey, subKey, now)
	if again.Encode() != signed.Encode() {
		t.Fatal("wbi signing must be deterministic")
	}
	// different ts -> different signature
	other := bilibiliSignWbi(params, imgKey, subKey, now+1)
	if other.Get("w_rid") == signed.Get("w_rid") {
		t.Fatal("w_rid should change with wts")
	}
}

func TestStripEmTags(t *testing.T) {
	in := `<em class="keyword">RSS</em>Hub 视频04`
	if got := stripEmTags(in); got != "RSSHub 视频04" {
		t.Fatalf("stripEmTags = %q", got)
	}
}

// TestBilibiliRoutesRegisterOnGin ensures every spec mounts on gin without
// wildcard/static conflicts or duplicate paths (a startup-time panic risk).
func TestBilibiliRoutesRegisterOnGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("gin route conflict: %v", p)
		}
	}()
	seen := make(map[string]bool, len(Routes))
	e := gin.New()
	for _, spec := range Routes {
		if seen[spec.Path] {
			t.Fatalf("duplicate route path %q", spec.Path)
		}
		seen[spec.Path] = true
		if spec.Handler == nil {
			t.Fatalf("route %q has no handler", spec.Path)
		}
		p := "/rss/bilibili/" + spec.Path
		e.GET(p, func(c *gin.Context) {})
	}
}
