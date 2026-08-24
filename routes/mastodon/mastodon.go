package routes

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var mastodonHostPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)

var mastodonAccountRoute = routeutils.RouteSpec{
	Path:        "account/:instance/:id",
	Name:        "Mastodon Account Statuses",
	Example:     "mastodon/account/mastodon.social/1",
	Maintainers: []string{"xihale"},
	Description: "Recent public statuses of a Mastodon account",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("instance", "Instance hostname, e.g. mastodon.social"),
		routeutils.RequiredParam("id", "Numeric account ID on that instance"),
		routeutils.OptionalParam("only_media", "Set to true to only return statuses with media"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  MastodonAccountHandler,
}

var mastodonTimelineRoute = routeutils.RouteSpec{
	Path:        "timeline/:instance",
	Name:        "Mastodon Public Timeline",
	Example:     "mastodon/timeline/fosstodon.org",
	Maintainers: []string{"xihale"},
	Description: "Public (federated or local) timeline of a Mastodon instance",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("instance", "Instance hostname, e.g. fosstodon.org"),
		routeutils.OptionalParam("local", "Set to true to only include local posts (default false)"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  MastodonTimelineHandler,
}

// MastodonAccountHandler handles /mastodon/account/:instance/:id
func MastodonAccountHandler(c *ctxpkg.Context) (*models.Feed, error) {
	instance, id := c.Param("instance"), c.Param("id")
	if !mastodonHostPattern.MatchString(instance) {
		return nil, fmt.Errorf("invalid instance %q", instance)
	}
	apiURL := fmt.Sprintf("https://%s/api/v1/accounts/%s/statuses?limit=20", instance, id)
	if routeutils.ParseBool(c.QueryParam("only_media"), false) {
		apiURL += "&only_media=true"
	}
	return fetchMastodonStatuses(c, apiURL, fmt.Sprintf("Mastodon account %s@%s", id, instance))
}

// MastodonTimelineHandler handles /mastodon/timeline/:instance
func MastodonTimelineHandler(c *ctxpkg.Context) (*models.Feed, error) {
	instance := c.Param("instance")
	if !mastodonHostPattern.MatchString(instance) {
		return nil, fmt.Errorf("invalid instance %q", instance)
	}
	apiURL := fmt.Sprintf("https://%s/api/v1/timelines/public?limit=20", instance)
	if routeutils.ParseBool(c.QueryParam("local"), false) {
		apiURL += "&local=true"
	}
	return fetchMastodonStatuses(c, apiURL, fmt.Sprintf("Mastodon public timeline (%s)", instance))
}

func fetchMastodonStatuses(c *ctxpkg.Context, apiURL, title string) (*models.Feed, error) {
	ctx := c.Parent()

	var statuses []mastodonStatus
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &statuses); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(title, "https://"+mastodonHostOf(apiURL), title)
	routeutils.AppendMappedItems(feed, statuses, 0, func(s mastodonStatus) *models.Item {
		return buildMastodonItem(s)
	})

	return feed, nil
}

func buildMastodonItem(s mastodonStatus) *models.Item {
	status := s
	titlePrefix := ""
	if status.Reblog != nil {
		titlePrefix = "Boost: "
		inner := status.Reblog
		status = *inner
	}

	content := strings.TrimSpace(status.Content)
	if content == "" {
		return nil
	}
	link := strings.TrimSpace(status.URL)
	if link == "" {
		link = status.URI
	}
	if link == "" {
		return nil
	}

	title := titlePrefix + truncateText(extractPlainText(content), 80)
	if title == "" || title == titlePrefix {
		title = titlePrefix + "Untitled post"
	}

	item := routeutils.NewItem(title, link, content, status.CreatedAt)
	item.GUID = status.ID
	if item.GUID == "" {
		item.GUID = link
	}
	name := status.Account.DisplayName
	if name == "" {
		name = status.Account.Acct
	}
	if name != "" {
		routeutils.SetAuthor(item, name, routeutils.WithAuthorURI(status.Account.URL))
	}
	if status.Sensitive {
		routeutils.SetCategories(item, "sensitive")
	}
	return item
}

// mastodonHostOf extracts the host from an API URL like https://host/api/...
func mastodonHostOf(apiURL string) string {
	rest := strings.TrimPrefix(apiURL, "https://")
	if idx := strings.Index(rest, "/"); idx >= 0 {
		return rest[:idx]
	}
	return rest
}

type mastodonStatus struct {
	ID        string          `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	URI       string          `json:"uri"`
	URL       string          `json:"url"`
	Sensitive bool            `json:"sensitive"`
	Content   string          `json:"content"`
	Account   mastodonAccount `json:"account"`
	Reblog    *mastodonStatus `json:"reblog"`
}

type mastodonAccount struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Acct        string `json:"acct"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
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
	text := strings.Join(strings.Fields(sb.String()), " ")
	return html.UnescapeString(text)
}
