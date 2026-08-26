package routes

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var gitHubCommentsRoute = routeutils.RouteSpec{
	Path:        "comments/:owner/:repo",
	Name:        "GitHub Repo Comments",
	Example:     "github/comments/DIYgod/RSSHub",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest issue and pull request comments across a GitHub repository",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
		routeutils.OptionalParam("number", "Issue or pull request number; omit for all recent comments"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitHubCommentsHandler,
}

var gitHubCommentsNumberRoute = routeutils.RouteSpec{
	Path:        "comments/:owner/:repo/:number",
	Name:        "GitHub Issue / Pull Request Comments",
	Example:     "github/comments/DIYgod/RSSHub/8116",
	Maintainers: []string{"xihale"},
	Description: "Fetch comments of a single GitHub issue or pull request (works with both)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("owner", "Repository owner"),
		routeutils.RequiredParam("repo", "Repository name"),
		routeutils.RequiredParam("number", "Issue or pull request number"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  GitHubCommentsHandler,
}

// GitHubCommentsHandler handles /github/comments/:owner/:repo[/:number].
// Issue and PR comments share the same API endpoints, so one route covers both.
func GitHubCommentsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	number := strings.TrimSpace(c.Param("number"))
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)

	if number == "" {
		url := fmt.Sprintf("%s/repos/%s/%s/issues/comments?sort=updated&direction=desc&per_page=%d",
			gitHubAPIBase, owner, repo, limit)
		return gitHubCommentsFeed(c, url, owner, repo, nil)
	}

	if routeutils.ParseOptionalPositiveInt(number) == nil {
		return nil, fmt.Errorf("invalid issue/pull number %q", number)
	}

	// A single number serves both issues and pull requests.
	topicURL := fmt.Sprintf("%s/repos/%s/%s/issues/%s", gitHubAPIBase, owner, repo, number)
	var issue GitHubIssue
	ctx := c.Parent()
	if err := gitHubGetJSON(ctx, c.Client(), topicURL, &issue); err != nil {
		return nil, fmt.Errorf("failed to fetch issue/pull %s of %s/%s: %w", number, owner, repo, err)
	}

	commentsURL := fmt.Sprintf("%s/comments?per_page=%d", topicURL, limit)
	return gitHubCommentsFeed(c, commentsURL, owner, repo, &issue)
}

// gitHubCommentsFeed builds the comments feed from an explicit comments URL.
// When issue is non-nil its opening post leads the feed as a timeline.
func gitHubCommentsFeed(c *ctxpkg.Context, commentsURL, owner, repo string, issue *GitHubIssue) (*models.Feed, error) {
	ctx := c.Parent()
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)

	var comments []GitHubComment
	if err := gitHubGetJSON(ctx, c.Client(), commentsURL, &comments); err != nil {
		return nil, err
	}

	feedTitle := fmt.Sprintf("%s/%s: Issue & Pull Request Comments", owner, repo)
	feedLink := fmt.Sprintf("%s/%s/%s/issues", gitHubWebBase, owner, repo)
	feed := routeutils.NewFeed(feedTitle, feedLink,
		fmt.Sprintf("Latest comments on %s/%s", owner, repo))

	if issue != nil {
		feed.Title = fmt.Sprintf("%s/%s: %s #%d - %s", owner, repo, commentTargetKind(issue), issue.Number, firstLine(issue.Title))
		feed.Link = issue.HTMLURL
		if feed.Link == "" {
			feed.Link = fmt.Sprintf("%s/%s/%s/issues/%d", gitHubWebBase, owner, repo, issue.Number)
		}

		rootItem := routeutils.NewItem(
			fmt.Sprintf("%s created %s/%s: %s #%d", authorLogin(issue.User), owner, repo, commentTargetKind(issue), issue.Number),
			feed.Link,
			excerptBody(issue.Body, 600),
			issue.CreatedAt,
		)
		if issue.Number != 0 {
			rootItem.GUID = fmt.Sprintf("gh-issue-%d", issue.Number)
		}
		if login := authorLogin(issue.User); login != "" {
			routeutils.SetItemAuthor(rootItem, login, "", fmt.Sprintf("%s/%s", gitHubWebBase, login))
		}
		routeutils.AddItem(feed, rootItem)
	}

	routeutils.AppendMappedItems(feed, comments, limit, func(comment GitHubComment) *models.Item {
		number, kind := resolveCommentTarget(owner, repo, comment)
		actor := comment.Actor.Login
		if actor == "" {
			actor = strings.TrimSpace(comment.User.Login)
		}
		if actor == "" {
			actor = "ghost"
		}
		item := routeutils.NewItem(
			fmt.Sprintf("%s commented on %s/%s: %s #%d", actor, owner, repo, kind, number),
			comment.HTMLURL,
			excerptBody(comment.Body, 600),
			comment.CreatedAt,
		)
		switch {
		case comment.ID != 0:
			item.GUID = fmt.Sprintf("gh-comment-%d", comment.ID)
		case comment.HTMLURL != "":
			item.GUID = comment.HTMLURL
		}
		routeutils.SetCategories(item, kind)
		if actor != "ghost" {
			routeutils.SetItemAuthor(item, actor, "", fmt.Sprintf("%s/%s", gitHubWebBase, actor))
		}
		return item
	})

	return feed, nil
}

// authorLogin extracts a safe display login from an API user object.
func authorLogin(user *GitHubUser) string {
	if user == nil || user.Login == "" {
		return "ghost"
	}
	return user.Login
}

// commentTargetKind classifies an issue object as an Issue or Pull Request.
func commentTargetKind(issue *GitHubIssue) string {
	if issue != nil && issue.PullRequest != nil {
		return "Pull Request"
	}
	return "Issue"
}

// resolveCommentTarget recovers the target issue number and type from a
// repository-wide comment (falls back to unknown values when absent).
func resolveCommentTarget(owner, repo string, comment GitHubComment) (int, string) {
	kind := "Issue"
	number := 0

	if comment.IssueURL == "" {
		return number, kind
	}
	prefix := fmt.Sprintf("%s/repos/%s/%s/issues/", gitHubAPIOrigin, owner, repo)
	parsed, err := strconv.Atoi(strings.TrimPrefix(comment.IssueURL, prefix))
	if err != nil {
		return number, kind
	}
	number = parsed

	if strings.Contains(comment.HTMLURL, "/pull/") {
		kind = "Pull Request"
	}
	return number, kind
}

// GitHubComment is one entry of the GitHub issue-comments endpoints.
type GitHubComment struct {
	ID        int       `json:"id"`
	HTMLURL   string    `json:"html_url"`
	IssueURL  string    `json:"issue_url"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Actor struct {
		Login string `json:"login"`
	} `json:"actor"`
}
