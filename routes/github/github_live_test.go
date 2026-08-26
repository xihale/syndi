package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestGitHubCommitsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubCommitsHandler, map[string]string{"owner": "gin-gonic", "repo": "gin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestGitHubIssuesLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubIssuesHandler, map[string]string{"owner": "golang", "repo": "go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	for _, item := range feed.Items {
		if item.Title == "" || item.Link == "" {
			t.Fatalf("item missing title/link: %+v", item)
		}
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestGitHubPullLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubPullHandler, map[string]string{"owner": "gin-gonic", "repo": "gin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestGitHubActivityLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubActivityHandler, map[string]string{"user": "torvalds"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	for _, item := range feed.Items {
		if item.Link == "" {
			t.Fatalf("item missing link: %+v", item)
		}
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestGitHubGistsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubGistsHandler, map[string]string{"user": "torvalds"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestGitHubStarsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	// Repo with releases.
	feed, err := testutil.RunHandler(GitHubStarsHandler, map[string]string{"owner": "gin-gonic", "repo": "gin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 || !strings.HasPrefix(feed.Items[0].GUID, "gh-release-") {
		t.Fatalf("unexpected releases feed: first=%+v", feed.Items[0])
	}
	if !strings.Contains(feed.Title, "Releases") {
		t.Fatalf("expected Releases title, got %q", feed.Title)
	}
	t.Logf("gin releases: %d items, first: %s", len(feed.Items), feed.Items[0].Title)

	// Repo without releases: tags fallback (torvalds/linux publishes only tags).
	tagFeed, err := testutil.RunHandler(GitHubStarsHandler, map[string]string{"owner": "torvalds", "repo": "linux"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tagFeed.Items) == 0 || !strings.HasPrefix(tagFeed.Items[0].GUID, "gh-release-") {
		t.Fatalf("tag fallback failed: first=%+v", tagFeed.Items[0])
	}
	if !strings.Contains(tagFeed.Title, "Tags") {
		t.Fatalf("expected Tags fallback title, got %q", tagFeed.Title)
	}
	t.Logf("linux tags: %d items, first: %s", len(tagFeed.Items), tagFeed.Items[0].Title)
}

func TestGitHubIssueByStateLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	for _, tc := range []struct{ path, state, labels string }{
		{"issue/:owner/:repo", "", ""},
		{"issue/:owner/:repo/closed", "closed", ""},
		{"issue/:owner/:repo/all/Performance", "all", "Performance"},
	} {
		feed, err := testutil.RunHandler(gitHubIssueHandler, map[string]string{
			"owner": "golang", "repo": "go", "state": tc.state, "labels": tc.labels,
		})
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		for _, item := range feed.Items {
			if !strings.HasPrefix(item.GUID, "gh-issue-") {
				t.Fatalf("%s: unexpected GUID %q", tc.path, item.GUID)
			}
		}
		t.Logf("%s: got %d items", tc.path, len(feed.Items))
	}
}

func TestGitHubPullByStateLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubPullHandler, map[string]string{
		"owner": "gin-gonic", "repo": "gin", "state": "closed",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range feed.Items {
		if !strings.HasPrefix(item.GUID, "gh-pull-") {
			t.Fatalf("unexpected GUID %q", item.GUID)
		}
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestGitHubCommentsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubCommentsHandler, map[string]string{
		"owner": "DIYgod", "repo": "RSSHub", "number": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected repo-wide comments")
	}
	t.Logf("repo-wide: got %d items, first: %s", len(feed.Items), feed.Items[0].Title)

	single, err := testutil.RunHandler(GitHubCommentsHandler, map[string]string{
		"owner": "DIYgod", "repo": "RSSHub", "number": "8116",
	})
	if err != nil {
		t.Fatalf("single issue comments: %v", err)
	}
	t.Logf("issue #8116: got %d items, title: %s", len(single.Items), single.Title)
}

func TestGitHubBranchesLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubBranchesHandler, map[string]string{"owner": "gin-gonic", "repo": "gin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected branches")
	}
	t.Logf("got %d branches, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestGitHubTopicsSearchLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubTopicsHandler, map[string]string{"name": "framework", "qs": ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected topic repositories")
	}
	t.Logf("got %d repos, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestGitHubFileCommitsLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(GitHubFileHandler, map[string]string{
		"owner":    "DIYgod",
		"repo":     "RSSHub",
		"branch":   "master",
		"filepath": "/README.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected file commits")
	}
	t.Logf("got %d commits, first: %s", len(feed.Items), feed.Items[0].Title)
}
