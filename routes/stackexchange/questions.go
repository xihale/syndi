package routes

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var stackExchangeQuestionsRoute = routeutils.RouteSpec{
	Path:        "questions/:site",
	Name:        "Stack Exchange Questions",
	Example:     "stackexchange/questions/stackoverflow",
	Maintainers: []string{"xihale"},
	Description: "Latest questions on a Stack Exchange site, sorted by activity",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("site", "Stack Exchange site (API site parameter), e.g. stackoverflow"),
		routeutils.OptionalParam("tagged", "Filter by tag, e.g. go"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  StackExchangeQuestionsHandler,
}

// StackExchangeQuestionsHandler handles /stackexchange/questions/:site
func StackExchangeQuestionsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	site := c.Param("site")
	tagged := c.QueryParam("tagged")
	ctx := c.Parent()

	apiURL := fmt.Sprintf(
		"https://api.stackexchange.com/2.3/questions?order=desc&sort=activity&site=%s&pagesize=30",
		url.QueryEscape(site),
	)
	if tagged != "" {
		apiURL += "&tagged=" + url.QueryEscape(tagged)
	}

	var resp seResponse
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Stack Exchange (%s)", site),
		fmt.Sprintf("https://%s/questions", siteAPIHost(site)),
		fmt.Sprintf("Latest questions on %s sorted by activity", site),
	)
	routeutils.AppendMappedItems(feed, resp.Items, 0, func(q seQuestion) *models.Item {
		if q.Title == "" || q.Link == "" {
			return nil
		}
		item := routeutils.NewItem(
			q.Title,
			q.Link,
			buildSEDescription(q),
			time.Unix(int64(q.CreationDate), 0),
		)
		item.GUID = fmt.Sprintf("se-%s-%d", site, q.QuestionID)
		if q.Owner.DisplayName != "" {
			routeutils.SetAuthor(item, q.Owner.DisplayName, routeutils.WithAuthorURI(q.Owner.Link))
		}
		routeutils.SetCategories(item, q.Tags...)
		return item
	})

	return feed, nil
}

func buildSEDescription(q seQuestion) string {
	var sb strings.Builder
	meta := fmt.Sprintf("Score: %d | Answers: %d | Views: %d", q.Score, q.AnswerCount, q.ViewCount)
	if q.IsAnswered {
		meta += " | Answered"
	}
	sb.WriteString(meta)
	if len(q.Tags) > 0 {
		sb.WriteString("<br/>Tags: " + html.EscapeString(strings.Join(q.Tags, ", ")))
	}
	return sb.String()
}

// siteAPIHost maps an API site parameter to its public web host.
func siteAPIHost(site string) string {
	switch site {
	case "stackoverflow":
		return "stackoverflow.com"
	case "serverfault":
		return "serverfault.com"
	case "superuser":
		return "superuser.com"
	case "askubuntu":
		return "askubuntu.com"
	default:
		return site + ".com"
	}
}

type seResponse struct {
	Items []seQuestion `json:"items"`
}

type seQuestion struct {
	Tags         []string `json:"tags"`
	Owner        seOwner  `json:"owner"`
	IsAnswered   bool     `json:"is_answered"`
	ViewCount    int      `json:"view_count"`
	AnswerCount  int      `json:"answer_count"`
	Score        int      `json:"score"`
	CreationDate int64    `json:"creation_date"`
	QuestionID   int64    `json:"question_id"`
	Link         string   `json:"link"`
	Title        string   `json:"title"`
}

type seOwner struct {
	DisplayName string `json:"display_name"`
	Link        string `json:"link"`
}
