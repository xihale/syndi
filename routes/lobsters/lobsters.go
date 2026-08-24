package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var lobstersHotRoute = routeutils.RouteSpec{
	Path:        "hot",
	Name:        "Lobsters Hot",
	Example:     "lobsters/hot",
	Maintainers: []string{"xihale"},
	Description: "Hottest stories on Lobsters",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    15 * time.Minute,
	Handler:     LobstersHotHandler,
}

var lobstersNewestRoute = routeutils.RouteSpec{
	Path:        "newest",
	Name:        "Lobsters Newest",
	Example:     "lobsters/newest",
	Maintainers: []string{"xihale"},
	Description: "Newest stories on Lobsters",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters:  nil,
	CacheTTL:    15 * time.Minute,
	Handler:     LobstersNewestHandler,
}

var lobstersTagRoute = routeutils.RouteSpec{
	Path:        "tag/:tag",
	Name:        "Lobsters Tag",
	Example:     "lobsters/tag/go",
	Maintainers: []string{"xihale"},
	Description: "Lobsters stories tagged with a specific tag",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("tag", "Tag name, e.g. golang"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  LobstersTagHandler,
}

// LobstersHotHandler handles /lobsters/hot
func LobstersHotHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return fetchLobsters(c, "https://lobste.rs/hottest.json", "Lobsters Hot")
}

// LobstersNewestHandler handles /lobsters/newest
func LobstersNewestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return fetchLobsters(c, "https://lobste.rs/newest.json", "Lobsters Newest")
}

// LobstersTagHandler handles /lobsters/tag/:tag
func LobstersTagHandler(c *ctxpkg.Context) (*models.Feed, error) {
	tag := c.Param("tag")
	url := fmt.Sprintf("https://lobste.rs/t/%s.json", url.PathEscape(tag))
	return fetchLobsters(c, url, fmt.Sprintf("Lobsters Tag: %s", tag))
}

func fetchLobsters(c *ctxpkg.Context, url, title string) (*models.Feed, error) {
	ctx := c.Parent()

	var stories []lobstersStory
	if err := routeutils.GetJSON(ctx, c.Client(), url, &stories); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(title, "https://lobste.rs/", "Lobsters: Computing news and discussion")
	routeutils.AppendMappedItems(feed, stories, 0, func(s lobstersStory) *models.Item {
		link := strings.TrimSpace(s.URL)
		if link == "" {
			link = s.CommentsURL
		}
		if link == "" {
			link = s.ShortIDURL
		}
		if s.Title == "" || link == "" {
			return nil
		}

		item := routeutils.NewItem(
			s.Title,
			link,
			buildLobstersDescription(s),
			s.CreatedAt,
		)
		item.GUID = s.ShortID
		if item.GUID == "" {
			item.GUID = link
		}
		if user := s.SubmitterUser.Username; user != "" {
			routeutils.SetAuthor(item, user, routeutils.WithAuthorURI("https://lobste.rs/u/"+user))
		}
		routeutils.SetCategories(item, s.Tags...)
		return item
	})

	return feed, nil
}

func buildLobstersDescription(s lobstersStory) string {
	var sb strings.Builder
	desc := strings.TrimSpace(s.Description)
	if desc != "" {
		sb.WriteString(desc)
	}
	var meta []string
	meta = append(meta, fmt.Sprintf("Score: %d", s.Score))
	meta = append(meta, fmt.Sprintf("Comments: %d", s.CommentCount))
	sb.WriteString("<br/>" + strings.Join(meta, " | "))
	if len(s.Tags) > 0 {
		sb.WriteString("<br/>Tags: " + html.EscapeString(strings.Join(s.Tags, ", ")))
	}
	return sb.String()
}

type lobstersStory struct {
	ShortID       string       `json:"short_id"`
	CreatedAt     time.Time    `json:"created_at"`
	Title         string       `json:"title"`
	URL           string       `json:"url"`
	Description   string       `json:"description"`
	Score         int          `json:"score"`
	CommentCount  int          `json:"comment_count"`
	SubmitterUser lobstersUser `json:"submitter_user"`
	Tags          []string     `json:"tags"`
	ShortIDURL    string       `json:"short_id_url"`
	CommentsURL   string       `json:"comments_url"`
}

// lobstersUser tolerates both API shapes: a plain username string or an object.
type lobstersUser struct {
	Username string
}

func (u *lobstersUser) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if trimmed[0] == '"' {
		var name string
		if err := json.Unmarshal(data, &name); err != nil {
			return err
		}
		u.Username = name
		return nil
	}
	var obj struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	u.Username = obj.Username
	return nil
}
