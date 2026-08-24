package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func zhihuLiveCookie(t *testing.T) {
	t.Helper()
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
}

func TestZhihuHotLive(t *testing.T) {
	zhihuLiveCookie(t)
	feed, err := testutil.RunHandler(ZhihuHotHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("hot: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestZhihuDailyLive(t *testing.T) {
	zhihuLiveCookie(t)
	feed, err := testutil.RunHandler(ZhihuDailyHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Items[0].Description == "" {
		t.Fatal("expected story body in description")
	}
	t.Logf("daily: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestZhihuDailySectionLive(t *testing.T) {
	zhihuLiveCookie(t)
	feed, err := testutil.RunHandler(ZhihuDailySectionHandler, map[string]string{"sectionId": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("daily section: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestZhihuZhuanlanLive(t *testing.T) {
	zhihuLiveCookie(t)
	if os.Getenv("ZHIHU_COOKIES") == "" {
		t.Skip("ZHIHU_COOKIES not set; zhuanlan requires login cookie")
	}
	feed, err := testutil.RunHandler(ZhihuZhuanlanHandler, map[string]string{"id": "googledevelopers"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Title == "" || feed.Title == "知乎专栏 - " {
		t.Fatalf("expected column meta title, got %q", feed.Title)
	}
	t.Logf("zhuanlan: %d items, feed=%s, first: %s", len(feed.Items), feed.Title, feed.Items[0].Title)
}

func TestZhihuQuestionLive(t *testing.T) {
	zhihuLiveCookie(t)
	if os.Getenv("ZHIHU_COOKIES") == "" {
		t.Skip("ZHIHU_COOKIES not set; question requires login cookie")
	}
	feed, err := testutil.RunHandler(ZhihuQuestionHandler, map[string]string{"questionId": "59895982"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("question: %d items, feed=%s, first: %s", len(feed.Items), feed.Title, feed.Items[0].Title)
}

func TestZhihuPeopleAnswersLive(t *testing.T) {
	zhihuLiveCookie(t)
	if os.Getenv("ZHIHU_COOKIES") == "" {
		t.Skip("ZHIHU_COOKIES not set; people answers requires login cookie")
	}
	feed, err := testutil.RunHandler(ZhihuPeopleAnswersHandler, map[string]string{"id": "diygod"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("people answers: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestZhihuPostsLive(t *testing.T) {
	zhihuLiveCookie(t)
	if os.Getenv("ZHIHU_COOKIES") == "" {
		t.Skip("ZHIHU_COOKIES not set; posts requires login cookie")
	}
	feed, err := testutil.RunHandler(ZhihuPostsHandler, map[string]string{"usertype": "people", "id": "frederchen"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("posts: %d items, feed=%s, first: %s", len(feed.Items), feed.Title, feed.Items[0].Title)
}

func TestZhihuActivitiesLive(t *testing.T) {
	zhihuLiveCookie(t)
	if os.Getenv("ZHIHU_COOKIES") == "" {
		t.Skip("ZHIHU_COOKIES not set; activities requires login cookie")
	}
	feed, err := testutil.RunHandler(ZhihuActivitiesHandler, map[string]string{"id": "kaifulee"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("activities: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestZhihuCollectionLive(t *testing.T) {
	zhihuLiveCookie(t)
	if os.Getenv("ZHIHU_COOKIES") == "" {
		t.Skip("ZHIHU_COOKIES not set; collection requires login cookie")
	}
	feed, err := testutil.RunHandler(ZhihuCollectionHandler, map[string]string{"id": "26444956"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("collection: %d items, feed=%s, first: %s", len(feed.Items), feed.Title, feed.Items[0].Title)
}

func TestZhihuWeeklyLive(t *testing.T) {
	zhihuLiveCookie(t)
	feed, err := testutil.RunHandler(ZhihuWeeklyHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("weekly: %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
