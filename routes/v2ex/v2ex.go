package routes

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const v2exBaseURL = "https://www.v2ex.com"

var v2exHotRoute = routeutils.RouteSpec{
	Path:        "hot",
	Name:        "V2EX Hot Topics",
	Example:     "v2ex/hot",
	Maintainers: []string{"xihale"},
	Description: "Today's hot topics on V2EX",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    15 * time.Minute,
	Handler:     V2EXHotHandler,
}

var v2exLatestRoute = routeutils.RouteSpec{
	Path:        "latest",
	Name:        "V2EX Latest Topics",
	Example:     "v2ex/latest",
	Maintainers: []string{"xihale"},
	Description: "Latest topics on V2EX",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    10 * time.Minute,
	Handler:     V2EXLatestHandler,
}

var v2exNodeRoute = routeutils.RouteSpec{
	Path:        "node/:name",
	Name:        "V2EX Node Topics",
	Example:     "v2ex/node/python",
	Maintainers: []string{"xihale"},
	Description: "Latest topics in a V2EX node",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("name", "Node name, e.g. python"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  V2EXNodeHandler,
}

var v2exTopicRoute = routeutils.RouteSpec{
	Path:        "topic/:id",
	Name:        "V2EX Topic Replies",
	Example:     "v2ex/topic/1",
	Maintainers: []string{"xihale"},
	Description: "Replies of a V2EX topic",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Numeric topic ID, e.g. 1"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  V2EXTopicHandler,
}

// V2EXHotHandler handles /v2ex/hot
func V2EXHotHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return fetchV2EXTopics(c, v2exBaseURL+"/api/topics/hot.json", "V2EX Hot Topics")
}

// V2EXLatestHandler handles /v2ex/latest
func V2EXLatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return fetchV2EXTopics(c, v2exBaseURL+"/api/topics/latest.json", "V2EX Latest Topics")
}

// V2EXNodeHandler handles /v2ex/node/:name
func V2EXNodeHandler(c *ctxpkg.Context) (*models.Feed, error) {
	name := c.Param("name")
	apiURL := fmt.Sprintf("%s/api/topics/show.json?node_name=%s", v2exBaseURL, url.QueryEscape(name))
	return fetchV2EXTopics(c, apiURL, fmt.Sprintf("V2EX Node: %s", name))
}

// V2EXTopicHandler handles /v2ex/topic/:id (replies of a topic)
func V2EXTopicHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	ctx := c.Parent()

	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		return nil, fmt.Errorf("invalid topic id %q: %w", id, err)
	}

	var topics []v2exTopic
	if err := routeutils.GetJSON(ctx, c.Client(), fmt.Sprintf("%s/api/topics/show.json?id=%s", v2exBaseURL, id), &topics); err != nil {
		return nil, err
	}
	if len(topics) == 0 {
		return nil, fmt.Errorf("topic %s not found", id)
	}
	topic := topics[0]
	topicURL := topic.URL
	if topicURL == "" {
		topicURL = fmt.Sprintf("%s/t/%s", v2exBaseURL, id)
	}

	var replies []v2exReply
	if err := routeutils.GetJSON(ctx, c.Client(), fmt.Sprintf("%s/api/replies/show.json?topic_id=%s", v2exBaseURL, id), &replies); err != nil {
		return nil, err
	}

	title := topic.Title
	if title == "" {
		title = "Topic " + id
	}
	feed := routeutils.NewFeed(
		fmt.Sprintf("V2EX Replies: %s", truncateText(title, 80)),
		topicURL,
		fmt.Sprintf("Replies of V2EX topic %s", id),
	)
	routeutils.AppendMappedItems(feed, replies, 0, func(r v2exReply) *models.Item {
		return buildV2EXReplyItem(topicURL, r)
	})

	return feed, nil
}

func buildV2EXReplyItem(topicURL string, r v2exReply) *models.Item {
	username := r.Member.Username
	content := strings.TrimSpace(r.ContentRendered)
	if content == "" && r.Content != "" {
		content = "<p>" + html.EscapeString(strings.TrimSpace(r.Content)) + "</p>"
	}
	if content == "" || topicURL == "" {
		return nil
	}
	title := truncateText(extractPlainText(content), 80)
	if title == "" {
		title = fmt.Sprintf("Reply by %s", username)
	}
	link := fmt.Sprintf("%s#reply-%d", topicURL, r.ID)
	item := routeutils.NewItem(
		title,
		link,
		content,
		time.Unix(int64(r.Created), 0),
	)
	if r.ID != 0 {
		item.GUID = fmt.Sprintf("v2ex-reply-%d", r.ID)
	}
	if username != "" {
		routeutils.SetAuthor(item, username, routeutils.WithAuthorURI(v2exBaseURL+"/u/"+username))
	}
	return item
}

func fetchV2EXTopics(c *ctxpkg.Context, apiURL, title string) (*models.Feed, error) {
	ctx := c.Parent()

	var topics []v2exTopic
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &topics); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(title, v2exBaseURL+"/", "Topics from V2EX")
	routeutils.AppendMappedItems(feed, topics, 0, func(t v2exTopic) *models.Item {
		link := t.URL
		if link == "" {
			link = fmt.Sprintf("%s/t/%d", v2exBaseURL, t.ID)
		}
		if t.Title == "" {
			return nil
		}
		description := t.ContentRendered
		if description == "" && t.Content != "" {
			description = "<p>" + html.EscapeString(t.Content) + "</p>"
		}
		description += fmt.Sprintf("<br/>Node: %s | Replies: %d", html.EscapeString(t.Node.Title), t.Replies)

		item := routeutils.NewItem(
			t.Title,
			link,
			description,
			time.Unix(int64(t.Created), 0),
		)
		item.GUID = fmt.Sprintf("v2ex-topic-%d", t.ID)
		if t.Member.Username != "" {
			routeutils.SetAuthor(item, t.Member.Username, routeutils.WithAuthorURI(v2exBaseURL+"/u/"+t.Member.Username))
		}
		if t.Node.Name != "" {
			routeutils.SetCategories(item, t.Node.Name)
		}
		return item
	})

	return feed, nil
}

type v2exTopic struct {
	ID              int      `json:"id"`
	Title           string   `json:"title"`
	URL             string   `json:"url"`
	Content         string   `json:"content"`
	ContentRendered string   `json:"content_rendered"`
	Replies         int      `json:"replies"`
	Created         int64    `json:"created"`
	Member          v2exUser `json:"member"`
	Node            v2exNode `json:"node"`
}

type v2exNode struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type v2exUser struct {
	Username string `json:"username"`
}

type v2exReply struct {
	ID              int      `json:"id"`
	Content         string   `json:"content"`
	ContentRendered string   `json:"content_rendered"`
	Created         int64    `json:"created"`
	Member          v2exUser `json:"member"`
}

// truncateText shortens plain text for use as a title.
func truncateText(s string, max int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// extractPlainText strips simple HTML tags from a fragment for titles.
func extractPlainText(s string) string {
	var sb strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			sb.WriteRune(r)
		}
	}
	text := sb.String()
	text = strings.Join(strings.Fields(text), " ")
	return html.UnescapeString(text)
}
