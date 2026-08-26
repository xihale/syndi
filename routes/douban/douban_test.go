package routes

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
)

// loadFixture reads a file from this package's testdata directory.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return data
}

func loadHTMLFixture(t *testing.T, name string) *parser.Document {
	t.Helper()
	doc, err := parser.LoadString(string(loadFixture(t, name)))
	if err != nil {
		t.Fatalf("failed to parse fixture %s: %v", name, err)
	}
	return doc
}

func loadJSONFixture(t *testing.T, name string, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(loadFixture(t, name), target); err != nil {
		t.Fatalf("failed to decode fixture %s: %v", name, err)
	}
}

func TestDoubanComingFixture(t *testing.T) {
	var resp doubanCollectionResp
	loadJSONFixture(t, "coming.json", &resp)

	feed := routeutils.NewFeed("豆瓣电影-即将上映", "https://movie.douban.com/coming", "")
	routeutils.AppendMappedItems(feed, resp.items(), 20, buildDoubanComingItem)
	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.Title != "肖申克的救赎" || first.GUID != "douban-coming-1292052" {
		t.Fatalf("unexpected first item: %+v", first)
	}
	if first.PubDate.Format("2006-01-02") != "2026-08-28" {
		t.Fatalf("expected pubdate from pubdate array, got %s", first.PubDate)
	}
	if len(first.Categories) != 2 || first.Categories[0] != "犯罪" {
		t.Fatalf("expected genres as categories, got %v", first.Categories)
	}
	if !strings.Contains(first.Description, `<li>导演：弗兰克·德拉邦特</li>`) ||
		!strings.Contains(first.Description, "<h2>剧情简介</h2>") {
		t.Fatalf("description missing info/intro blocks: %s", first.Description)
	}
}

func TestDoubanLaterFixture(t *testing.T) {
	doc := loadHTMLFixture(t, "later.html")
	feed := routeutils.NewFeed("即将上映的电影", "https://movie.douban.com/cinema/later/", "")
	doubanAppendLaterItems(feed, doc)

	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if !strings.Contains(first.Title, "08月28日 - 《肖申克的救赎》 - 犯罪 / 剧情") {
		t.Fatalf("unexpected title %q", first.Title)
	}
	if first.GUID != "douban-later-1292052" || first.Link != "https://movie.douban.com/subject/1292052/" {
		t.Fatalf("unexpected guid/link: %s %s", first.GUID, first.Link)
	}
	if !strings.Contains(first.Description, "想看人数：519304") {
		t.Fatalf("description missing want count: %s", first.Description)
	}
}

func TestDoubanWeeklyCollectionFixture(t *testing.T) {
	var resp doubanCollectionResp
	loadJSONFixture(t, "weekly.json", &resp)

	feed := routeutils.NewFeed("豆瓣电影一周口碑榜", "", "")
	doubanAppendCollectionItems(feed, resp.items())
	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.Title != "蜘蛛侠：崭新之日" || first.GUID != "douban-subject-36246195" {
		t.Fatalf("unexpected first item: %+v", first)
	}
	if !strings.Contains(first.Description, "7.8 分（90210 人评）") {
		t.Fatalf("rating missing in description: %s", first.Description)
	}
	if second := feed.Items[1]; !strings.Contains(second.Description, "评价人数不足") {
		t.Fatalf("null rating reason missing: %s", second.Description)
	}
}

func TestDoubanUSBoxFixture(t *testing.T) {
	var resp doubanUSBoxResp
	loadJSONFixture(t, "usbox.json", &resp)
	if len(resp.Subjects) != 2 {
		t.Fatalf("expected 2 subjects, got %d", len(resp.Subjects))
	}

	item := buildDoubanUSBoxItem(resp.Subjects[0].Rank, resp.Subjects[0].Box, resp.Subjects[0].Subject)
	if item == nil {
		t.Fatal("expected item")
	}
	if item.GUID != "douban-ustop-36246195" {
		t.Fatalf("unexpected guid %q", item.GUID)
	}
	for _, want := range []string{"第 1 名", "周末票房：3900万", "评分：7.8"} {
		if !strings.Contains(item.Description, want) {
			t.Fatalf("description missing %q: %s", want, item.Description)
		}
	}

	noRating := buildDoubanUSBoxItem(resp.Subjects[1].Rank, resp.Subjects[1].Box, resp.Subjects[1].Subject)
	if !strings.Contains(noRating.Description, "评分：无") {
		t.Fatalf("unstarred rating should render 无: %s", noRating.Description)
	}
}

func TestDoubanGroupFixture(t *testing.T) {
	doc := loadHTMLFixture(t, "group.html")

	if name := groupName(doc); name != "可爱事物分享组" {
		t.Fatalf("unexpected group name %q", name)
	}
	feed := routeutils.NewFeed("豆瓣小组", "https://www.douban.com/group/648102/", "")
	doubanAppendGroupItems(feed, doc)

	if len(feed.Items) != 3 {
		t.Fatalf("expected 3 items (header row skipped), got %d", len(feed.Items))
	}
	pinned := feed.Items[0]
	if pinned.Title != "可爱事物分享组管理条例" {
		t.Fatalf("title attribute should win over link text: %q", pinned.Title)
	}
	if pinned.GUID != "douban-group-220964802" {
		t.Fatalf("unexpected guid %q", pinned.GUID)
	}
	if got := routeutils.GetAuthorString(&pinned); got != "4399" {
		t.Fatalf("unexpected author %q", got)
	}
	if !strings.Contains(pinned.Description, "回复：94") {
		t.Fatalf("reply count missing: %s", pinned.Description)
	}
}

func TestDoubanDoulistFixture(t *testing.T) {
	doc := loadHTMLFixture(t, "doulist.html")

	feed := routeutils.NewFeed(
		routeutils.CollapseWhitespace(doc.First("#content h1").TextTrim()),
		"https://www.douban.com/doulist/37716774/",
		"",
	)
	doubanAppendDoulistItems(feed, doc)
	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}

	note := feed.Items[0]
	if note.Title != "推荐一些好习惯，或将改变你的人生" || note.GUID != "douban-doulist-225401867" {
		t.Fatalf("unexpected note item: %+v", note)
	}
	if note.PubDate.Format("2006-01-02 15:04") != "2015-08-21 10:07" {
		t.Fatalf("timestamp not parsed from time text: %s", note.PubDate)
	}

	subject := feed.Items[1]
	if subject.Title != "肖申克的救赎" || subject.GUID != "douban-doulist-228110290" {
		t.Fatalf("unexpected subject item: %+v", subject)
	}
	if subject.PubDate.Format("2006-01-02 15:04") != "2015-08-20 12:00" {
		t.Fatalf("timestamp not parsed from span title: %s", subject.PubDate)
	}
	if !strings.Contains(subject.Description, `<blockquote>`) || !strings.Contains(subject.Description, `<img width="100"`) {
		t.Fatalf("subject description missing poster/comment: %s", subject.Description)
	}
}

func TestDoubanTopicFixture(t *testing.T) {
	var resp doubanTopicResp
	loadJSONFixture(t, "topic.json", &resp)

	feed, err := buildDoubanTopicFeed(context.Background(), nil, "48823",
		"https://www.douban.com/gallery/topic/48823/?sort=new", resp,
		func(ctx context.Context, cl *client.Client, id string) (string, error) {
			return "<div>笔记全文</div>", nil
		})
	if err != nil {
		t.Fatalf("failed to build topic feed: %v", err)
	}
	if feed.Title != "令你难忘的深夜长谈-豆瓣话题" {
		t.Fatalf("unexpected feed title %q", feed.Title)
	}
	if len(feed.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(feed.Items))
	}

	status := feed.Items[0]
	if status.Title != "深夜用户的广播" || status.GUID != "douban-topic-330011223" {
		t.Fatalf("unexpected status item: %+v", status)
	}
	if !strings.Contains(status.Description, "<br>") || !strings.Contains(status.Description, `<img src="https://img1.doubanio.com/view/status/l/public/2b33f8ac7a95a58.jpg"/>`) {
		t.Fatalf("status description missing line break/image: %s", status.Description)
	}
	if got := routeutils.GetAuthorString(&status); got != "深夜用户" {
		t.Fatalf("unexpected author %q", got)
	}

	post := feed.Items[1]
	if post.Title != "和陌生人的通宵长谈" || post.GUID != "douban-topic-149988210" {
		t.Fatalf("unexpected topic item: %+v", post)
	}

	note := feed.Items[2]
	if note.Description != "<div>笔记全文</div>" {
		t.Fatalf("note should use fetched full text, got: %s", note.Description)
	}
}

func TestDoubanExploreFixture(t *testing.T) {
	doc := loadHTMLFixture(t, "explore.html")
	feed := routeutils.NewFeed("豆瓣-浏览发现", "https://www.douban.com/explore/recommend", "")
	doubanAppendExploreItems(feed, doc)

	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.Title != "发了个朋友圈被客户嘲笑了" || first.GUID != "douban-explore-497332688" {
		t.Fatalf("unexpected first item: %+v", first)
	}
	if !strings.Contains(first.Description, `<img src="https://qnmob3.doubanio.com/view/group_topic/large/public/p743569619.jpg?imageView2/1/q/60/w/300/h/300/format/jpg"/>`) {
		t.Fatalf("cover image missing: %s", first.Description)
	}
	if got := routeutils.GetAuthorString(&feed.Items[1]); got != "招财猫" {
		t.Fatalf("unexpected author %q", got)
	}
	if !strings.Contains(feed.Items[1].Description, "锯子刨子都上了") {
		t.Fatalf("description text missing: %s", feed.Items[1].Description)
	}
}

func TestDoubanBookLatestFixture(t *testing.T) {
	var resp doubanCollectionResp
	loadJSONFixture(t, "booklatest.json", &resp)

	feed := routeutils.NewFeed("豆瓣新书速递", "https://book.douban.com/latest", "")
	doubanAppendBookItems(feed, resp.items())
	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.GUID != "douban-book-38556395" || !strings.Contains(first.Description, "8.6 分（78 人评）") {
		t.Fatalf("unexpected book item: %+v", first)
	}
	if !strings.Contains(first.Description, "奥伯斯特多夫村") {
		t.Fatal("cards content missing in description")
	}
	if second := feed.Items[1]; !strings.Contains(second.Description, "评价人数不足") {
		t.Fatalf("null rating fallback missing: %s", second.Description)
	}
}

func TestDoubanMusicLatestHTMLFixture(t *testing.T) {
	doc := loadHTMLFixture(t, "musiclatest.html")
	feed := routeutils.NewFeed("豆瓣最新增加的音乐", "https://music.douban.com/latest", "")
	doubanAppendMusicLatestHTML(feed, doc)

	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.Title != "Never Do" || first.GUID != "douban-music-latest-38621368" {
		t.Fatalf("unexpected first item: %+v", first)
	}
	if !strings.Contains(first.Description, "charlieonnafriday / 2026-02-06 / 单曲 / 数字(Digital)") {
		t.Fatalf("info line missing: %s", first.Description)
	}
}

func TestDoubanMusicAreaFixture(t *testing.T) {
	var resp doubanCollectionResp
	loadJSONFixture(t, "musicarea.json", &resp)

	feed := routeutils.NewFeed("豆瓣最新增加的音乐-华语新碟榜", "", "")
	doubanAppendMusicCollection(feed, resp.items())
	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.Title != "快乐有时有-蔡旻佑/流行/2026-08-21" || first.GUID != "douban-music-38602233" {
		t.Fatalf("unexpected music item: %+v", first)
	}
	if first.PubDate.Format("2006-01-02") != "2026-08-21" {
		t.Fatalf("pubdate not parsed: %s", first.PubDate)
	}
	if !strings.Contains(first.Description, "时隔多年再次发片的诚意之作") || !strings.Contains(first.Description, "7.2 分（108 人评）") {
		t.Fatalf("comment/rating missing: %s", first.Description)
	}
}

func TestDoubanEventHotFixture(t *testing.T) {
	var resp doubanCollectionResp
	loadJSONFixture(t, "eventhot.json", &resp)

	items := resp.items()
	if len(items) != 1 {
		t.Fatalf("expected 1 event, got %d", len(items))
	}
	item := buildDoubanEventItem(items[0])
	if item == nil {
		t.Fatal("expected item")
	}
	if item.GUID != "douban-event-108877441" {
		t.Fatalf("unexpected guid %q", item.GUID)
	}
	if !strings.Contains(item.Description, "音乐") || !strings.Contains(item.Description, "180 - 680元") {
		t.Fatalf("event meta missing: %s", item.Description)
	}
}

func TestDoubanTVComingFixture(t *testing.T) {
	var resp doubanCollectionResp
	loadJSONFixture(t, "tvcoming.json", &resp)

	subjects := resp.items()
	if len(subjects) != 2 {
		t.Fatalf("expected 2 subjects, got %d", len(subjects))
	}
	item := buildDoubanTVComingItem(subjects[0])
	if item == nil {
		t.Fatal("expected item")
	}
	if item.GUID != "douban-tv-coming-37045704" {
		t.Fatalf("unexpected guid %q", item.GUID)
	}
	if item.PubDate.Format("2006-01-02") != "2026-11-28" {
		t.Fatalf("pubdate not parsed: %s", item.PubDate)
	}
	if len(item.Categories) != 2 {
		t.Fatalf("genres should become categories: %v", item.Categories)
	}
	if second := buildDoubanTVComingItem(subjects[1]); second == nil || second.Link != "https://movie.douban.com/subject/36509090/" {
		t.Fatalf("missing url should fall back to subject link: %+v", second)
	}
}

func TestDoubanJobsFixture(t *testing.T) {
	doc := loadHTMLFixture(t, "jobs.html")
	feed := routeutils.NewFeed("豆瓣社会招聘", "https://jobs.douban.com/jobs/social/", "")
	doubanAppendJobItems(feed, doc, "social")

	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.Title != "运维开发工程师" || first.GUID != "douban-jobs-position-ywkfgcs" {
		t.Fatalf("unexpected job item: %+v", first)
	}
	if first.Link != "https://jobs.douban.com/jobs/social/#position-ywkfgcs" {
		t.Fatalf("anchor link wrong: %s", first.Link)
	}
	if !strings.Contains(first.Description, "<strong>职位描述:</strong>") {
		t.Fatalf("section headers missing: %s", first.Description)
	}
}

func TestDoubanChannelFixture(t *testing.T) {
	var info doubanChannelInfoResp
	loadJSONFixture(t, "channel_info.json", &info)
	if info.Title != "书法" {
		t.Fatalf("channel title decoded wrong: %q", info.Title)
	}

	var resp doubanChannelFeedResp
	loadJSONFixture(t, "channel_feed.json", &resp)
	if len(resp.Items) != 2 {
		t.Fatalf("fixture decode failed, got %d raw items", len(resp.Items))
	}

	if item := buildDoubanChannelItem(resp.Items[0]); item == nil {
		t.Fatal("first channel item should be kept")
	} else {
		if item.Title != "开始学书法 看这一篇就够了" || item.GUID != "douban-channel-605477098" {
			t.Fatalf("unexpected channel item: %+v", item)
		}
		if got := routeutils.GetAuthorString(item); got != "雲胡不喜" {
			t.Fatalf("unexpected author %q", got)
		}
	}
	if item := buildDoubanChannelItem(resp.Items[1]); item != nil {
		t.Fatalf("aggregated card must be filtered out, got %+v", item)
	}
}

func TestDoubanSanitizeKeyAndIDFromLink(t *testing.T) {
	if got := doubanSanitizeKey("movie_weekly_best", "movie_weekly_best"); got != "movie_weekly_best" {
		t.Fatalf("simple key rejected: %q", got)
	}
	if got := doubanSanitizeKey("../etc/passwd", "fallback"); got != "fallback" {
		t.Fatalf("path traversal not sanitized: %q", got)
	}
	if got := doubanIDFromLink("https://movie.douban.com/subject/1292052/?from=tag"); got != "1292052" {
		t.Fatalf("id extraction broken: %q", got)
	}
	if got := doubanIDFromLink(""); got != "" {
		t.Fatalf("empty link should give empty id: %q", got)
	}
}

func TestDoubanAPIErrorEnvelope(t *testing.T) {
	err := decodeDoubanJSON("https://m.douban.com/rexxar/api/v2/test", []byte(`{"code":103,"msg":"need_login","request":"GET /v2/x"}`), &struct{}{})
	if err == nil {
		t.Fatal("error envelope must surface an error")
	}
	doubanErr, ok := err.(*doubanAPIError)
	if !ok || doubanErr.Code != 103 {
		t.Fatalf("expected doubanAPIError, got %#v", err)
	}

	var payload struct {
		Title string `json:"title"`
	}
	if err := decodeDoubanJSON("https://x", []byte(`{"code":0,"title":"ok"}`), &payload); err != nil || payload.Title != "ok" {
		t.Fatalf("normal payload should decode, got %v %q", err, payload.Title)
	}
}

func TestDoubanFormatBox(t *testing.T) {
	cases := map[float64]string{
		39_000_000:  "3900万",
		230_000_000: "2.30亿",
		999:         "999元",
	}
	for input, want := range cases {
		if got := doubanFormatBox(input); got != want {
			t.Fatalf("doubanFormatBox(%f) = %q, want %q", input, got, want)
		}
	}
}
