package routes

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var doubanTopicRoute = routeutils.RouteSpec{
	Path:        "topic/:id",
	Name:        "Douban Topic",
	Example:     "douban/topic/48823",
	Maintainers: []string{"xihale"},
	Description: "Posts of a douban gallery topic (豆瓣话题)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Topic id, e.g. 48823"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  DoubanTopicHandler,
}

var doubanTopicSortRoute = routeutils.RouteSpec{
	Path:        "topic/:id/:sort",
	Name:        "Douban Topic by Sort",
	Example:     "douban/topic/48823/hot",
	Maintainers: []string{"xihale"},
	Description: "Posts of a douban gallery topic (豆瓣话题)",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Topic id, e.g. 48823"),
		routeutils.RequiredParam("sort", "Sort order, hot or new, default new"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  DoubanTopicHandler,
}

// doubanNoteFetcher fetches the full HTML of a topic note entry.
type doubanNoteFetcher func(ctx context.Context, cl *client.Client, noteID string) (string, error)

type doubanTopicResp struct {
	Items []doubanTopicItem `json:"items"`
}

type doubanTopicItem struct {
	Target doubanTopicTarget `json:"target"`
	Topic  *doubanTopicInfo  `json:"topic"`
}

type doubanTopicInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Introduction string `json:"introduction"`
}

type doubanTopicTarget struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	Abstract   string `json:"abstract"`
	SharingURL string `json:"sharing_url"`
	CreateTime string `json:"create_time"`

	Author struct {
		Name string `json:"name"`
	} `json:"author"`

	Status struct {
		SharingURL string `json:"sharing_url"`
		Text       string `json:"text"`
		CreateTime string `json:"create_time"`
		Author     struct {
			Name string `json:"name"`
		} `json:"author"`
		Images []struct {
			Normal struct {
				URL string `json:"url"`
			} `json:"normal"`
		} `json:"images"`
	} `json:"status"`

	Photos []struct {
		Src string `json:"src"`
	} `json:"photos"`
}

// DoubanTopicHandler handles /douban/topic/:id/:sort?
//
// The rexxar gallery API started requiring a logged-in cookie for topic item
// listing; without one it answers need_login. The route stays faithful to
// upstream and works with account cookies.
func DoubanTopicHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := doubanSanitizeKey(c.Param("id"), "")
	if id == "" {
		return nil, fmt.Errorf("douban topic: invalid id %q", c.Param("id"))
	}
	sort := routeutils.ParseEnum(c.Param("sort"), "new", "hot", "new")
	ctx := c.Parent()
	cl := c.Client()

	apiURL := fmt.Sprintf("%s/gallery/topic/%s/items?sort=%s&start=0&count=10&status_full_text=1", doubanRexxarAPI, id, sort)
	link := fmt.Sprintf("%s/gallery/topic/%s/?sort=%s", doubanWWWBaseURL, id, sort)
	var resp doubanTopicResp
	if err := doubanFetchJSON(ctx, cl, apiURL, link, &resp); err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("douban topic: no items for topic %s", id)
	}

	noteFull := doubanNoteFetcher(fetchDoubanNoteHTML)
	return buildDoubanTopicFeed(ctx, cl, id, link, resp, noteFull)
}

// buildDoubanTopicFeed assembles the feed from an already decoded payload;
// split out so fixture tests can exercise the parsing logic.
func buildDoubanTopicFeed(ctx context.Context, cl *client.Client, id, link string, resp doubanTopicResp, noteFull doubanNoteFetcher) (*models.Feed, error) {
	title := id
	description := ""
	for i := range resp.Items {
		if resp.Items[i].Topic != nil && resp.Items[i].Topic.Name != "" {
			title = resp.Items[i].Topic.Name
			description = resp.Items[i].Topic.Introduction
			break
		}
	}

	feed := routeutils.NewFeed(fmt.Sprintf("%s-豆瓣话题", title), link, description)
	for _, raw := range resp.Items {
		item := doubanTopicFeedItem(raw.Target, link, noteFull, ctx, cl)
		if item != nil {
			routeutils.AddItem(feed, item)
		}
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("douban topic: no usable items in payload for topic %s", id)
	}
	return feed, nil
}

// doubanTopicFeedItem renders one topic entry (status broadcast, topic post,
// or note). Any unknown shape falls back to whatever fields exist.
func doubanTopicFeedItem(target doubanTopicTarget, fallbackLink string, noteFull doubanNoteFetcher, ctx context.Context, cl *client.Client) *models.Item {
	var (
		author      string
		title       string
		link        string
		dateText    string
		description string
		key         string // guid suffix
	)

	switch target.Type {
	case "status":
		status := target.Status
		author = status.Author.Name
		link = strings.SplitN(status.SharingURL, "&", 2)[0]
		title = author + "的广播"
		dateText = status.CreateTime

		text := html.EscapeString(status.Text)
		text = strings.ReplaceAll(text, "\n", "<br>")
		description = text
		for _, img := range status.Images {
			if img.Normal.URL != "" {
				description += fmt.Sprintf(`<br><img src="%s"/>`, html.EscapeString(img.Normal.URL))
			}
		}
	case "note", "review":
		author = target.Author.Name
		link = firstNonEmpty(target.SharingURL, fallbackLink)
		title = author + "的日记"
		dateText = target.CreateTime
		key = target.ID
		description = html.EscapeString(routeutils.CollapseWhitespace(target.Abstract))
		if key != "" && noteFull != nil {
			if full, err := noteFull(ctx, cl, key); err == nil && full != "" {
				description = full
			}
		}
	default:
		author = target.Author.Name
		link = firstNonEmpty(target.SharingURL, fallbackLink)
		title = firstNonEmpty(routeutils.CollapseWhitespace(target.Title), author+"的话题")
		dateText = target.CreateTime

		text := html.EscapeString(routeutils.CollapseWhitespace(target.Abstract))
		description = text
		for _, photo := range target.Photos {
			if photo.Src != "" {
				description += fmt.Sprintf(`<br><img src="%s"/>`, html.EscapeString(photo.Src))
			}
		}
	}

	if title == "" || link == "" || description == "" {
		return nil
	}

	item := routeutils.NewItem(title, link, description, doubanParseDate(dateText))
	if item == nil {
		return nil
	}
	if author != "" {
		routeutils.SetAuthor(item, author)
	}
	guidKey := firstNonEmpty(key, doubanIDFromLink(link), title)
	item.GUID = "douban-topic-" + guidKey
	return item
}

// fetchDoubanNoteHTML loads the server-rendered full text of a note.
func fetchDoubanNoteHTML(ctx context.Context, cl *client.Client, noteID string) (string, error) {
	noteURL := fmt.Sprintf("%s/j/note/%s/full", doubanWWWBaseURL, noteID)
	raw, err := doubanWebProfile().Referer(doubanWWWBaseURL+"/").Fetch(noteURL).GetString(ctx, cl)
	if err != nil {
		return "", err
	}
	cleaned, err := routeutils.CleanDescription(raw, doubanWWWBaseURL+"/", routeutils.CleanOptions{
		Sanitize:       true,
		BaseURL:        doubanWWWBaseURL + "/",
		ReferrerPolicy: true,
		RemoveScripts:  true,
		FixLazyImages:  true,
		ResolveLinks:   true,
		AllowedTags:    []string{"p", "br", "b", "strong", "em", "i", "u", "blockquote", "img", "a", "div", "span"},
	})
	if err != nil {
		return "", err
	}
	// Wrap in a div tag so downstream renderers keep it as rich content.
	return "<div>" + cleaned + "</div>", nil
}
