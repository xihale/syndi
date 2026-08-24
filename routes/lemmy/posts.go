package routes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var lemmyHostPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)

var lemmyPostsRoute = routeutils.RouteSpec{
	Path:        "posts/:instance",
	Name:        "Lemmy Posts",
	Example:     "lemmy/posts/lemmy.world",
	Maintainers: []string{"xihale"},
	Description: "Hot posts across a Lemmy instance's federated front page",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("instance", "Instance hostname, e.g. lemmy.world"),
		routeutils.OptionalParam("sort", "Sort order: Hot (default), Active, New, TopDay, TopWeek, MostComments"),
		routeutils.OptionalParam("community", "Limit to a community name on the instance"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  LemmyPostsHandler,
}

// LemmyPostsHandler handles /lemmy/posts/:instance
func LemmyPostsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	instance := c.Param("instance")
	if !lemmyHostPattern.MatchString(instance) {
		return nil, fmt.Errorf("invalid instance %q", instance)
	}
	sort := routeutils.ParseEnum(c.QueryParam("sort"), "Hot",
		"Hot", "Active", "New", "TopDay", "TopWeek", "TopMonth", "TopAll", "MostComments")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 50)
	ctx := c.Parent()

	apiURL := fmt.Sprintf("https://%s/api/v3/post/list?limit=%d&sort=%s", instance, limit, sort)
	if community := c.QueryParam("community"); community != "" {
		apiURL += "&community_name=" + urlEscape(community)
	}

	var resp lemmyPostListResponse
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Lemmy (%s)", instance),
		fmt.Sprintf("https://%s/", instance),
		fmt.Sprintf("Posts on %s sorted by %s", instance, sort),
	)
	routeutils.AppendMappedItems(feed, resp.Posts, 0, func(entry lemmyPostView) *models.Item {
		return buildLemmyItem(instance, entry)
	})

	return feed, nil
}

func buildLemmyItem(instance string, entry lemmyPostView) *models.Item {
	post := entry.Post
	if post.Name == "" {
		return nil
	}
	link := strings.TrimSpace(post.URL)
	if link == "" {
		link = post.APID
	}
	if link == "" {
		return nil
	}

	description := strings.TrimSpace(post.BodyHTML)
	if description == "" {
		if body := strings.TrimSpace(post.Body); body != "" {
			description = "<p>" + htmlEscapeText(body) + "</p>"
		} else if embed := strings.TrimSpace(post.EmbedDescription); embed != "" {
			description = "<p>" + htmlEscapeText(embed) + "</p>"
		}
	}
	var meta []string
	if entry.Community.Title != "" {
		meta = append(meta, "Community: "+htmlEscapeText(entry.Community.Title))
	}
	meta = append(meta, fmt.Sprintf("Score: %d | Comments: %d", entry.Counts.Score, entry.Counts.Comments))
	if description == "" {
		description = strings.Join(meta, " | ")
	} else {
		description += "<br/>" + strings.Join(meta, " | ")
	}

	item := routeutils.NewItem(post.Name, link, description, post.Published)
	item.GUID = fmt.Sprintf("lemmy-%s-%d", instance, post.ID)
	if author := authorName(entry.Creator); author != "" {
		routeutils.SetAuthor(item, author, routeutils.WithAuthorURI(entry.Creator.ActorID))
	}
	if entry.Nsfw || post.Nsfw {
		routeutils.SetCategories(item, "NSFW")
	}
	if entry.Community.Name != "" {
		routeutils.SetCategories(item, entry.Community.Name)
	}
	return item
}

func authorName(creator lemmyPerson) string {
	if creator.DisplayName != "" {
		return creator.DisplayName
	}
	return creator.Name
}

func urlEscape(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '~':
			sb.WriteRune(r)
		default:
			for _, b := range []byte(string(r)) {
				sb.WriteString(fmt.Sprintf("%%%02X", b))
			}
		}
	}
	return sb.String()
}

func htmlEscapeText(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

type lemmyPostListResponse struct {
	Posts []lemmyPostView `json:"posts"`
}

type lemmyPostView struct {
	Post      lemmyPost      `json:"post"`
	Creator   lemmyPerson    `json:"creator"`
	Community lemmyCommunity `json:"community"`
	Counts    lemmyCounts    `json:"counts"`
	Nsfw      bool           `json:"nsfw"`
}

type lemmyPost struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	URL              string    `json:"url"`
	Body             string    `json:"body"`
	BodyHTML         string    `json:"body_html"`
	EmbedDescription string    `json:"embed_description"`
	Published        time.Time `json:"published"`
	Nsfw             bool      `json:"nsfw"`
	APID             string    `json:"ap_id"`
}

type lemmyPerson struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	ActorID     string `json:"actor_id"`
}

type lemmyCommunity struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

type lemmyCounts struct {
	Score    int `json:"score"`
	Comments int `json:"comments"`
}
