package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/disguise"
	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

const linuxdoBaseURL = "https://linux.do"

// linux.do sits behind a Cloudflare managed challenge that blocks the default
// client fingerprint; a minimal consistent XHR fingerprint (verified against
// the live site) gets the Discourse JSON through.
var linuxdoProfile = disguise.Custom(
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
).
	Lang("zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7").
	WithHeader("Accept", "application/json, text/javascript, */*; q=0.01").
	WithHeader("X-Requested-With", "XMLHttpRequest")

var linuxdoLatestRoute = routeutils.RouteSpec{
	Path:        "latest",
	Name:        "LINUX DO Latest Topics",
	Example:     "linuxdo/latest",
	Maintainers: []string{"xihale"},
	Description: "Latest topics on LINUX DO (Discourse forum)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters:  nil,
	CacheTTL:    15 * time.Minute,
	Handler:     LinuxDoLatestHandler,
}

var linuxdoTopRoute = routeutils.RouteSpec{
	Path:        "top",
	Name:        "LINUX DO Top Topics",
	Example:     "linuxdo/top",
	Maintainers: []string{"xihale"},
	Description: "Top topics on LINUX DO (Discourse forum)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("period", "Top period: all (default), yearly, quarterly, monthly, weekly, daily"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  LinuxDoTopHandler,
}

// LinuxDoLatestHandler handles /linuxdo/latest
func LinuxDoLatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return fetchLinuxDo(c, linuxdoBaseURL+"/latest.json", "LINUX DO Latest")
}

// LinuxDoTopHandler handles /linuxdo/top
func LinuxDoTopHandler(c *ctxpkg.Context) (*models.Feed, error) {
	period := routeutils.ParseEnum(c.QueryParam("period"), "", "all", "yearly", "quarterly", "monthly", "weekly", "daily")
	url := linuxdoBaseURL + "/top.json"
	if period != "" && period != "all" {
		url += "?period=" + period
	}
	return fetchLinuxDo(c, url, "LINUX DO Top")
}

func fetchLinuxDo(c *ctxpkg.Context, apiURL, title string) (*models.Feed, error) {
	ctx := c.Parent()

	var resp discourseResponse
	if err := linuxdoProfile.Fetch(apiURL).GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(title, linuxdoBaseURL+"/", "Topics from LINUX DO")
	routeutils.AppendMappedItems(feed, resp.TopicList.Topics, 0, func(t discourseTopic) *models.Item {
		if t.Title == "" || t.ID == 0 {
			return nil
		}
		link := fmt.Sprintf("%s/t/%s/%d", linuxdoBaseURL, t.Slug, t.ID)
		meta := fmt.Sprintf("Replies: %d | Likes: %d", maxInt(t.PostsCount-1, 0), t.LikeCount)
		if !t.LastPostedAt.IsZero() && t.LastPostedAt.After(t.CreatedAt) {
			meta += fmt.Sprintf(" | Last reply: %s", t.LastPostedAt.UTC().Format(time.RFC3339))
		}
		item := routeutils.NewItem(
			t.Title,
			link,
			htmlEscapeString(t.Title)+"<br/>"+meta,
			t.CreatedAt,
		)
		item.GUID = fmt.Sprintf("linuxdo-topic-%d", t.ID)
		return item
	})

	return feed, nil
}

type discourseResponse struct {
	TopicList struct {
		Topics []discourseTopic `json:"topics"`
	} `json:"topic_list"`
}

type discourseTopic struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Slug         string    `json:"slug"`
	CreatedAt    time.Time `json:"created_at"`
	LastPostedAt time.Time `json:"last_posted_at"`
	PostsCount   int       `json:"posts_count"`
	LikeCount    int       `json:"like_count"`
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func htmlEscapeString(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			sb.WriteString("&amp;")
		case '<':
			sb.WriteString("&lt;")
		case '>':
			sb.WriteString("&gt;")
		case '"':
			sb.WriteString("&#34;")
		case '\'':
			sb.WriteString("&#39;")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
