package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xihale/syndi/internal/client"
	ctxpkg "github.com/xihale/syndi/pkg/context"
)

// fixtureServer serves files from routes/github/testdata keyed by exact
// request paths and records every request it receives.
type fixtureServer struct {
	*httptest.Server
	mu       sync.Mutex
	routes   map[string]string // path -> testdata file name
	requests []*http.Request
}

func newFixtureServer(t *testing.T, routes map[string]string) *fixtureServer {
	t.Helper()
	fs := &fixtureServer{routes: routes}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, ok := fs.routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		reqCopy := r.Clone(r.Context())
		reqCopy.RequestURI = ""
		fs.mu.Lock()
		fs.requests = append(fs.requests, reqCopy)
		fs.mu.Unlock()

		data, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Errorf("read fixture %s: %v", file, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(fs.Close)
	return fs
}

func (fs *fixtureServer) lastQuery(t *testing.T) string {
	t.Helper()
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if len(fs.requests) == 0 {
		t.Fatal("no requests received")
	}
	return fs.requests[len(fs.requests)-1].URL.RawQuery
}

// withGitHubAPIBase points the GitHub API base at baseURL for the duration of fn.
func withGitHubAPIBase(t *testing.T, baseURL string, fn func()) {
	t.Helper()
	orig := gitHubAPIBase
	gitHubAPIBase = baseURL
	defer func() { gitHubAPIBase = orig }()
	fn()
}

func routeContext(params map[string]string) *ctxpkg.Context {
	req := httptest.NewRequest(http.MethodGet, "/rss/test", nil)
	rec := httptest.NewRecorder()
	c := ctxpkg.NewContext(rec, req)
	c.SetParams(params)
	c.SetClient(client.New(client.WithTimeout(5 * time.Second)))
	return c
}

func TestGitHubStarsReleasesFeed(t *testing.T) {
	fs := newFixtureServer(t, map[string]string{
		"/repos/octo/hello/releases": "stars_releases.json",
	})
	withGitHubAPIBase(t, fs.URL, func() {
		feed, err := GitHubStarsHandler(routeContext(map[string]string{"owner": "octo", "repo": "hello"}))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(feed.Items); n != 2 {
			t.Fatalf("expected 2 items, got %d", n)
		}
		first := feed.Items[0]
		if first.Title != "Release v1.10.0" || first.Link == "" {
			t.Fatalf("unexpected first item: %+v", first)
		}
		if first.GUID != "gh-release-v1.10.0" {
			t.Fatalf("expected gh-release- GUID prefix, got %q", first.GUID)
		}
		if !strings.Contains(first.Description, "shiny feature") {
			t.Fatalf("release body missing: %q", first.Description)
		}
		if first.PubDate.IsZero() {
			t.Fatal("expected release pub date to be set")
		}
	})
}

func TestGitHubStarsTagsFallback(t *testing.T) {
	fs := newFixtureServer(t, map[string]string{
		"/repos/octo/empty/releases": "empty_releases.json",
		"/repos/octo/empty/tags":     "stars_tags.json",
	})
	withGitHubAPIBase(t, fs.URL, func() {
		feed, err := GitHubStarsHandler(routeContext(map[string]string{"owner": "octo", "repo": "empty"}))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(feed.Items); n != 2 {
			t.Fatalf("expected 2 tag items, got %d", n)
		}
		for _, item := range feed.Items {
			wantGUID := "gh-release-" + strings.TrimPrefix(item.Title, "Tag ")
			if item.GUID != wantGUID {
				t.Fatalf("tag GUID mismatch: got %q want %q", item.GUID, wantGUID)
			}
			if !strings.Contains(item.Link, "/tree/") {
				t.Fatalf("tag link should point at tree view: %q", item.Link)
			}
		}
	})
}

func TestGitHubIssuesAndPullsSplitSameFixture(t *testing.T) {
	fs := newFixtureServer(t, map[string]string{
		"/repos/octo/hello/issues": "issues_list.json",
	})

	withGitHubAPIBase(t, fs.URL, func() {
		issuesFeed, err := gitHubIssueHandler(routeContext(map[string]string{"owner": "octo", "repo": "hello", "state": "open"}))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(issuesFeed.Items); n != 2 {
			t.Fatalf("expected 2 issues after PR filtering, got %d", n)
		}
		for _, item := range issuesFeed.Items {
			if !strings.HasPrefix(item.GUID, "gh-issue-") {
				t.Fatalf("issue GUID prefix mismatch: %q", item.GUID)
			}
			if strings.Contains(item.Title, "documentation build") {
				t.Fatalf("pull request leaked into issue feed")
			}
			if item.Author == nil || item.Author.Name == "" {
				t.Fatalf("issue item missing author: %+v", item)
			}
		}

		pullFeed, err := GitHubPullHandler(routeContext(map[string]string{"owner": "octo", "repo": "hello", "state": "open"}))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(pullFeed.Items); n != 2 {
			t.Fatalf("expected 2 pulls, got %d", n)
		}
		for _, item := range pullFeed.Items {
			if !strings.HasPrefix(item.GUID, "gh-pull-") {
				t.Fatalf("pull GUID prefix mismatch: %q", item.GUID)
			}
		}

		if q := fs.lastQuery(t); q == "" || !strings.Contains(q, "state=open") || !strings.Contains(q, "sort=created") {
			t.Fatalf("unexpected issues query: %q", q)
		}
	})
}

func TestGitHubIssueLabelsForwardedToAPI(t *testing.T) {
	fs := newFixtureServer(t, map[string]string{
		"/repos/octo/hello/issues": "issues_list.json",
	})
	withGitHubAPIBase(t, fs.URL, func() {
		if _, err := gitHubIssueHandler(routeContext(map[string]string{"owner": "octo", "repo": "hello", "state": "closed", "labels": "bug,priority"})); err != nil {
			t.Fatal(err)
		}
		q := fs.lastQuery(t)
		if !strings.Contains(q, "labels=bug%2Cpriority") || !strings.Contains(q, "state=closed") {
			t.Fatalf("labels/state not forwarded: %q", q)
		}
	})
}

func TestGitHubCommentsRepoWide(t *testing.T) {
	fs := newFixtureServer(t, map[string]string{
		"/repos/octo/hello/issues/comments": "comments_all.json",
	})
	withGitHubAPIBase(t, fs.URL, func() {
		feed, err := GitHubCommentsHandler(routeContext(map[string]string{"owner": "octo", "repo": "hello"}))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(feed.Items); n != 2 {
			t.Fatalf("expected 2 comments, got %d", n)
		}
		first, second := feed.Items[0], feed.Items[1]
		if first.Title != "alice commented on octo/hello: Issue #42" {
			t.Fatalf("unexpected comment title: %q", first.Title)
		}
		if second.Title != "review-bot commented on octo/hello: Pull Request #7" {
			t.Fatalf("PR comment should be classified as Pull Request: %q", second.Title)
		}
		if first.GUID != "gh-comment-9000001" {
			t.Fatalf("comment GUID mismatch: %q", first.GUID)
		}
		for _, item := range feed.Items {
			if len(item.Categories) == 0 || (item.Categories[0] != "Issue" && item.Categories[0] != "Pull Request") {
				t.Fatalf("expected comment kind category, got %v", item.Categories)
			}
		}
		if q := fs.lastQuery(t); !strings.Contains(q, "sort=updated") {
			t.Fatalf("repo-wide comments should sort by updated, got %q", q)
		}
	})
}

func TestGitHubCommentsSingleNumber(t *testing.T) {
	fs := newFixtureServer(t, map[string]string{
		"/repos/octo/world/issues/42":          "issue_detail.json",
		"/repos/octo/world/issues/42/comments": "comments_issue.json",
	})
	withGitHubAPIBase(t, fs.URL, func() {
		feed, err := GitHubCommentsHandler(routeContext(map[string]string{"owner": "octo", "repo": "world", "number": "42"}))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(feed.Items); n != 3 {
			t.Fatalf("expected root post + 2 comments, got %d", n)
		}
		root := feed.Items[0]
		if root.GUID != "gh-issue-42" {
			t.Fatalf("root post GUID mismatch: %q", root.GUID)
		}
		if strings.Count(feed.Title, "#42") != 1 {
			t.Fatalf("feed title should reference issue #42: %q", feed.Title)
		}
		last := feed.Items[len(feed.Items)-1]
		if last.Categories[0] != "Issue" {
			t.Fatalf("single-issue comments must be Issue-typed: %+v", last)
		}
		if strings.Contains(last.Description, "<script>") {
			t.Fatalf("description must be escaped: %q", last.Description)
		}

		if _, err := GitHubCommentsHandler(routeContext(map[string]string{"owner": "octo", "repo": "world", "number": "not-a-number"})); err == nil {
			t.Fatal("expected error for non-numeric issue number")
		}
	})
}

func TestGitHubBranchesFeed(t *testing.T) {
	fs := newFixtureServer(t, map[string]string{
		"/repos/octo/hello/branches": "branches.json",
	})
	withGitHubAPIBase(t, fs.URL, func() {
		feed, err := GitHubBranchesHandler(routeContext(map[string]string{"owner": "octo", "repo": "hello"}))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(feed.Items); n != 3 {
			t.Fatalf("expected 3 named branches, got %d", n)
		}
		names := []string{feed.Items[0].Title, feed.Items[1].Title, feed.Items[2].Title}
		want := []string{"main", "feature/next-gen", "release-2.x"}
		for i := range want {
			if names[i] != want[i] {
				t.Fatalf("branch order mismatch: %v", names)
			}
			if feed.Items[i].Link == "" {
				t.Fatalf("branch link missing: %+v", feed.Items[i])
			}
		}
		if !strings.Contains(feed.Items[0].Description, "1111111") {
			t.Fatalf("expected short HEAD sha in description: %q", feed.Items[0].Description)
		}
	})
}

func TestGitHubTopicsSearchFeed(t *testing.T) {
	fs := newFixtureServer(t, map[string]string{
		"/search/repositories": "topics_search.json",
	})
	withGitHubAPIBase(t, fs.URL, func() {
		feed, err := GitHubTopicsHandler(routeContext(map[string]string{"name": "framework", "qs": "l=go&s=forks&o=asc"}))
		if err != nil {
			t.Fatal(err)
		}
		q := fs.lastQuery(t)
		if !strings.Contains(q, "topic%3Aframework") || !strings.Contains(q, "language%3Ago") ||
			!strings.Contains(q, "sort=forks") || !strings.Contains(q, "order=asc") {
			t.Fatalf("search query not built as expected: %q", q)
		}

		if n := len(feed.Items); n != 2 {
			t.Fatalf("expected 2 search items, got %d", n)
		}
		first := feed.Items[0]
		if first.Title != "acme/great-framework" {
			t.Fatalf("unexpected title: %q", first.Title)
		}
		if !strings.Contains(first.Description, "Stars: 12300") {
			t.Fatalf("unexpected description: %q", first.Description)
		}
		// language + up to 5 topic categories
		if len(first.Categories) != 6 {
			t.Fatalf("expected language+5 topics categories, got %v", first.Categories)
		}
		if feed.Items[1].PubDate.IsZero() {
			t.Fatalf("expected updated_at fallback date for pushed_at-less repo")
		}
	})
}

func TestGitHubFileCommitsFeed(t *testing.T) {
	fs := newFixtureServer(t, map[string]string{
		"/repos/octo/hello/commits": "file_commits.json",
	})
	withGitHubAPIBase(t, fs.URL, func() {
		feed, err := GitHubFileHandler(routeContext(map[string]string{
			"owner":    "octo",
			"repo":     "hello",
			"branch":   "main",
			"filepath": "/docs/guide/intro.md",
		}))
		if err != nil {
			t.Fatal(err)
		}

		q := fs.lastQuery(t)
		if !strings.Contains(q, "sha=main") || !strings.Contains(q, "path=docs%2Fguide%2Fintro.md") {
			t.Fatalf("file commits query wrong: %q", q)
		}

		if n := len(feed.Items); n != 2 {
			t.Fatalf("expected 2 commits, got %d", n)
		}
		first := feed.Items[0]
		if first.Title != "docs: rewrite install section" {
			t.Fatalf("commit title should be first message line: %q", first.Title)
		}
		if !strings.HasPrefix(first.Description, "<pre>") {
			t.Fatalf("description should be escaped pre block: %q", first.Description)
		}
		if first.PubDate.IsZero() {
			t.Fatal("committer date expected as pub date")
		}
		if second := feed.Items[1]; second.PubDate.IsZero() {
			t.Fatal("author date fallback expected when committer date missing")
		}

		if _, err := GitHubFileHandler(routeContext(map[string]string{
			"owner": "octo", "repo": "hello", "branch": "main", "filepath": "",
		})); err == nil {
			t.Fatal("expected error for empty filepath")
		}
	})
}
