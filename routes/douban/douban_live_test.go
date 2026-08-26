package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
	"github.com/xihale/syndi/pkg/models"
)

// Live tests hit real douban endpoints; they are skipped unless LIVE=1.
func wantLive(t *testing.T) {
	t.Helper()
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
}

func checkFeed(t *testing.T, feed *models.Feed, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatalf("expected items for feed %q", feed.Title)
	}
	first := feed.Items[0]
	if first.Title == "" || first.Link == "" || first.GUID == "" {
		t.Fatalf("expected title/link/guid, got %q %q %q", first.Title, first.Link, first.GUID)
	}
	t.Logf("%s: got %d items, first: %s (%s)", feed.Title, len(feed.Items), first.Title, first.Link)
}

func TestDoubanMoviePlayingLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanMoviePlayingHandler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestDoubanMoviePlayingFilteredLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanMoviePlayingHandler, map[string]string{"score": "8.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d filtered items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestDoubanMovieComingLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanMovieComingHandler, nil)
	checkFeed(t, feed, err)
}

func TestDoubanMovieLaterLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanMovieLaterHandler, nil)
	checkFeed(t, feed, err)
}

func TestDoubanMovieWeeklyLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanMovieWeeklyHandler, nil)
	checkFeed(t, feed, err)
}

func TestDoubanMovieWeeklyTypeLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanMovieWeeklyHandler, map[string]string{"type": "tv_chinese_best_weekly"})
	checkFeed(t, feed, err)
}

func TestDoubanMovieUSBoxLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanMovieUSBoxHandler, nil)
	checkFeed(t, feed, err)
}

func TestDoubanGroupLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanGroupHandler, map[string]string{"groupid": "648102"})
	checkFeed(t, feed, err)
}

func TestDoubanGroupTypeLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanGroupHandler, map[string]string{"groupid": "648102", "type": "essence"})
	checkFeed(t, feed, err)
}

func TestDoubanDoulistLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanDoulistHandler, map[string]string{"id": "37716774"})
	checkFeed(t, feed, err)
}

func TestDoubanTopicLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanTopicHandler, map[string]string{"id": "48823"})
	if err != nil {
		// The rexxar gallery API now requires a logged-in cookie for item
		// listing; surface it but do not fail hard.
		t.Skipf("topic API likely requires login: %v", err)
	}
	checkFeed(t, feed, nil)
}

func TestDoubanExploreLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanExploreHandler, nil)
	checkFeed(t, feed, err)
}

func TestDoubanBookLatestLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanBookLatestHandler, nil)
	checkFeed(t, feed, err)
}

func TestDoubanBookLatestTypeLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanBookLatestHandler, map[string]string{"type": "fiction"})
	checkFeed(t, feed, err)
}

func TestDoubanMusicLatestLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanMusicLatestHandler, nil)
	checkFeed(t, feed, err)
}

func TestDoubanMusicLatestAreaLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanMusicLatestHandler, map[string]string{"area": "chinese"})
	checkFeed(t, feed, err)
}

func TestDoubanEventHotLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanEventHotHandler, map[string]string{"locationId": "118172"})
	checkFeed(t, feed, err)
}

func TestDoubanTVComingLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanTVComingHandler, nil)
	checkFeed(t, feed, err)

	timeFeed, err := testutil.RunHandler(DoubanTVComingHandler, map[string]string{"sortBy": "time"})
	checkFeed(t, timeFeed, err)
}

func TestDoubanJobsLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanJobsHandler, map[string]string{"type": "social"})
	checkFeed(t, feed, err)
}

func TestDoubanChannelLive(t *testing.T) {
	wantLive(t)
	feed, err := testutil.RunHandler(DoubanChannelHandler, map[string]string{"id": "30168934", "nav": "hot"})
	checkFeed(t, feed, err)
}
