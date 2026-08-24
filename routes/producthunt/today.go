package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

const phHomeURL = "https://www.producthunt.com/"

// phProfile disguises requests against the Product Hunt homepage (SSR payload).
var phProfile = disguise.Chrome()

var (
	phEventsRe  = regexp.MustCompile(`"events":(\[.+\])\}\)`)
	phScriptRe  = regexp.MustCompile(`(?s)<script[^>]*>(.*?)</script>`)
	phUndefined = regexp.MustCompile(`\bundefined\b`)
)

type phEvent struct {
	Type  string       `json:"type"`
	Value *phEventBody `json:"value"`
}

type phEventBody struct {
	Data struct {
		Homefeed *phHomefeed `json:"homefeed"`
	} `json:"data"`
}

type phHomefeed struct {
	Edges []struct {
		Node phFeedNode `json:"node"`
	} `json:"edges"`
}

type phFeedNode struct {
	ID    string `json:"id"`
	Items []struct {
		Typename string `json:"__typename"`
		ID       string `json:"id"`
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		Tagline  string `json:"tagline"`
		Product  struct {
			Slug string `json:"slug"`
		} `json:"product"`
		ThumbnailImageUUID string `json:"thumbnailImageUuid"`
		CreatedAt          string `json:"createdAt"`
		Topics             struct {
			Edges []struct {
				Node struct {
					Name string `json:"name"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"topics"`
	} `json:"items"`
}

var todayRoute = routeutils.RouteSpec{
	Path:        "today",
	Name:        "Top Products Launching Today",
	Example:     "producthunt/today",
	Maintainers: []string{"xihale"},
	Description: "Top products launching today on Product Hunt",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    30 * time.Minute,
	Handler:     TodayHandler,
}

// TodayHandler handles /producthunt/today
func TodayHandler(c *ctxpkg.Context) (*models.Feed, error) {
	body, err := phProfile.Fetch(phHomeURL).GetString(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}

	node, err := extractPHFeaturedNode(body)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("Product Hunt Today Popular", phHomeURL, "Top products launching today on Product Hunt")

	for _, raw := range node.Items {
		if raw.Typename != "Post" || raw.Name == "" {
			continue
		}
		link := fmt.Sprintf("https://www.producthunt.com/products/%s", raw.Product.Slug)
		if raw.Product.Slug == "" && raw.Slug != "" {
			link = fmt.Sprintf("https://www.producthunt.com/posts/%s", raw.Slug)
		}

		var desc strings.Builder
		if raw.Tagline != "" {
			desc.WriteString("<p>" + html.EscapeString(raw.Tagline) + "</p>")
		}
		if raw.ThumbnailImageUUID != "" {
			desc.WriteString(fmt.Sprintf(`<img src="https://ph-files.imgix.net/%s" alt=""><br/>`, raw.ThumbnailImageUUID))
		}

		pubDate, _ := dateutil.ParseDate(raw.CreatedAt)

		item := routeutils.NewItem(raw.Name, link, desc.String(), pubDate)
		if raw.ID != "" {
			item.GUID = "ph-post-" + raw.ID
		}
		var topics []string
		for _, te := range raw.Topics.Edges {
			if te.Node.Name != "" {
				topics = append(topics, te.Node.Name)
			}
		}
		routeutils.SetCategories(item, topics...)
		routeutils.AddItem(feed, item)
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("producthunt: no posts found in home feed SSR payload")
	}
	return feed, nil
}

// extractPHFeaturedNode pulls the FEATURED-0 (today's launches) node out of the
// Apollo SSR data transport embedded in homepage scripts.
func extractPHFeaturedNode(body string) (*phFeedNode, error) {
	candidates := []string{}
	var scripts strings.Builder
	for _, m := range phScriptRe.FindAllStringSubmatch(body, -1) {
		if strings.Contains(m[1], "ApolloSSRDataTransport") {
			scripts.WriteString(m[1])
		}
	}
	scriptsText := scripts.String()
	if scriptsText != "" {
		candidates = append(candidates, scriptsText)
	}
	candidates = append(candidates, body)

	for _, candidate := range candidates {
		m := phEventsRe.FindStringSubmatch(candidate)
		if m == nil {
			continue
		}
		raw := strings.TrimSpace(m[1])
		raw = phUndefined.ReplaceAllString(raw, "null")
		var events []phEvent
		if err := json.Unmarshal([]byte(raw), &events); err != nil {
			continue
		}
		for i := range events {
			ev := &events[i]
			if ev.Type != "next" || ev.Value == nil || ev.Value.Data.Homefeed == nil {
				continue
			}
			for _, edge := range ev.Value.Data.Homefeed.Edges {
				if edge.Node.ID == "FEATURED-0" {
					n := edge.Node
					return &n, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("producthunt: ApolloSSRDataTransport homefeed payload not found")
}
