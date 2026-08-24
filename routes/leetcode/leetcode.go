package routes

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const (
	leetCodeCNBaseURL = "https://leetcode.cn"
	leetCodeENBaseURL = "https://leetcode.com"
)

// Routes lists all LeetCode route specs in this package.
var Routes = []routeutils.RouteSpec{
	leetCodeDailyCNRoute,
	leetCodeDailyENRoute,
}

var leetCodeDailyCNRoute = routeutils.RouteSpec{
	Path:        "dailyquestion/cn",
	Name:        "LeetCode China Daily Question",
	Example:     "leetcode/dailyquestion/cn",
	Maintainers: []string{"xihale"},
	Description: "Daily coding challenge question on leetcode.cn with problem description and topic tags",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     LeetCodeDailyQuestionCNHandler,
}

var leetCodeDailyENRoute = routeutils.RouteSpec{
	Path:        "dailyquestion/en",
	Name:        "LeetCode Daily Question",
	Example:     "leetcode/dailyquestion/en",
	Maintainers: []string{"xihale"},
	Description: "Daily coding challenge question on leetcode.com with problem description and topic tags",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     LeetCodeDailyQuestionENHandler,
}

// LeetCodeDailyQuestionCNHandler handles /leetcode/dailyquestion/cn.
func LeetCodeDailyQuestionCNHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()
	endpoint := leetCodeCNBaseURL + "/graphql"

	var daily lcTodayRecordResp
	query := `query questionOfToday { todayRecord { date question { frontendQuestionId: questionFrontendId titleSlug } } }`
	if err := leetCodeGraphQL(ctx, c.Client(), endpoint, query, map[string]any{}, &daily); err != nil {
		return nil, err
	}
	if len(daily.Data.TodayRecord) == 0 {
		return nil, fmt.Errorf("leetcode.cn returned no daily question record")
	}
	rec := daily.Data.TodayRecord[0]
	slug := rec.Question.TitleSlug
	if slug == "" {
		return nil, fmt.Errorf("leetcode.cn daily record has no titleSlug")
	}

	var detail lcQuestionDetailResp
	detailQuery := `query questionData($titleSlug: String!) {
		question(titleSlug: $titleSlug) {
			questionFrontendId
			translatedTitle
			translatedContent
			difficulty
			topicTags { name slug translatedName }
		}
	}`
	if err := leetCodeGraphQL(ctx, c.Client(), endpoint, detailQuery,
		map[string]any{"titleSlug": slug}, &detail); err != nil {
		return nil, err
	}
	q := detail.Data.Question
	if q.TranslatedTitle == "" && q.Content() == "" {
		return nil, fmt.Errorf("leetcode.cn returned no detail for %s", slug)
	}

	title := q.TranslatedTitle
	if title == "" {
		title = slug
	}
	link := fmt.Sprintf("%s/problems/%s/", leetCodeCNBaseURL, slug)
	desc := lcBuildDescription(q.Difficulty, q.Content(), q.TopicTags)

	pubDate, _ := time.Parse("2006-01-02", rec.Date)
	feed := routeutils.NewFeed("LeetCode 每日一题", leetCodeCNBaseURL, "LeetCode 中国站每日一题")
	item := routeutils.NewItem(fmt.Sprintf("%s. %s", q.FrontendID.String(), title), link, desc, pubDate)
	item.GUID = "leetcode-cn-daily-" + slug
	routeutils.AddItem(feed, item)
	return feed, nil
}

// LeetCodeDailyQuestionENHandler handles /leetcode/dailyquestion/en.
func LeetCodeDailyQuestionENHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx := c.Parent()
	endpoint := leetCodeENBaseURL + "/graphql"

	var daily lcActiveChallengeResp
	query := `query questionOfToday { activeDailyCodingChallengeQuestion { date link question { frontendQuestionId: questionFrontendId titleSlug } } }`
	if err := leetCodeGraphQL(ctx, c.Client(), endpoint, query, map[string]any{}, &daily); err != nil {
		return nil, err
	}
	challenge := daily.Data.ActiveDailyCodingChallengeQuestion
	if challenge.Question.TitleSlug == "" {
		return nil, fmt.Errorf("leetcode.com returned no daily question record")
	}
	slug := challenge.Question.TitleSlug

	var detail lcQuestionDetailResp
	detailQuery := `query questionData($titleSlug: String!) {
		question(titleSlug: $titleSlug) {
			questionFrontendId
			title
			content
			difficulty
			topicTags { name slug }
		}
	}`
	if err := leetCodeGraphQL(ctx, c.Client(), endpoint, detailQuery,
		map[string]any{"titleSlug": slug}, &detail); err != nil {
		return nil, err
	}
	q := detail.Data.Question
	if q.Title == "" && q.Content() == "" {
		return nil, fmt.Errorf("leetcode.com returned no detail for %s", slug)
	}

	title := q.Title
	if title == "" {
		title = slug
	}
	link := challenge.Link
	if link == "" {
		link = "/problems/" + slug + "/"
	}
	if !strings.HasPrefix(link, "http") {
		link = leetCodeENBaseURL + link
	}
	desc := lcBuildDescription(q.Difficulty, q.Content(), q.TopicTags)

	pubDate, _ := time.Parse("2006-01-02", challenge.Date)
	feed := routeutils.NewFeed("LeetCode Daily Question", leetCodeENBaseURL, "LeetCode daily coding challenge")
	item := routeutils.NewItem(fmt.Sprintf("%s. %s", q.FrontendID.String(), title), link, desc, pubDate)
	item.GUID = "leetcode-en-daily-" + slug
	routeutils.AddItem(feed, item)
	return feed, nil
}

// leetCodeGraphQL posts a GraphQL query to a LeetCode endpoint.
func leetCodeGraphQL(ctx context.Context, cl *client.Client, endpoint, query string, variables map[string]any, target any) error {
	body := map[string]any{"query": query, "variables": variables}
	return disguise.Chrome().JSONAccept().PostJSON(endpoint, body).GetJSON(ctx, cl, target)
}

// lcBuildDescription renders difficulty and tags metadata followed by the
// upstream HTML problem statement.
func lcBuildDescription(difficulty, contentHTML string, tags []lcTopicTag) string {
	var sb strings.Builder
	sb.WriteString("<p>")
	switch difficulty {
	case "Easy":
		sb.WriteString("🟢 ")
	case "Medium":
		sb.WriteString("🟡 ")
	case "Hard":
		sb.WriteString("🔴 ")
	}
	sb.WriteString(html.EscapeString(difficulty))
	if len(tags) > 0 {
		names := make([]string, 0, len(tags))
		for _, t := range tags {
			name := t.TranslatedName
			if name == "" {
				name = t.Name
			}
			if name == "" {
				continue
			}
			names = append(names, html.EscapeString(name))
		}
		if len(names) > 0 {
			sb.WriteString(" | 标签: ")
			sb.WriteString(strings.Join(names, ", "))
		}
	}
	sb.WriteString("</p><hr/>")
	sb.WriteString(contentHTML)
	return sb.String()
}

// lcTodayRecordResp is the payload of the CN daily-question GraphQL query.
type lcTodayRecordResp struct {
	Data struct {
		TodayRecord []struct {
			Date     string `json:"date"`
			Question struct {
				FrontendID lcFlexString `json:"frontendQuestionId"`
				TitleSlug  string       `json:"titleSlug"`
			} `json:"question"`
		} `json:"todayRecord"`
	} `json:"data"`
}

// lcActiveChallengeResp is the payload of the EN daily-question GraphQL query.
type lcActiveChallengeResp struct {
	Data struct {
		ActiveDailyCodingChallengeQuestion struct {
			Date     string `json:"date"`
			Link     string `json:"link"`
			Question struct {
				FrontendID lcFlexString `json:"frontendQuestionId"`
				TitleSlug  string       `json:"titleSlug"`
			} `json:"question"`
		} `json:"activeDailyCodingChallengeQuestion"`
	} `json:"data"`
}

// lcTopicTag is one topic tag of a question.
type lcTopicTag struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	TranslatedName string `json:"translatedName"`
}

// lcQuestion is one LeetCode question detail record.
type lcQuestion struct {
	FrontendID        lcFlexString `json:"questionFrontendId"`
	Title             string       `json:"title"`
	TranslatedTitle   string       `json:"translatedTitle"`
	RawContent        string       `json:"content"`
	TranslatedContent string       `json:"translatedContent"`
	Difficulty        string       `json:"difficulty"`
	TopicTags         []lcTopicTag `json:"topicTags"`
}

// Content returns whichever content flavor is populated (translated or original).
func (q *lcQuestion) Content() string {
	if q.TranslatedContent != "" {
		return q.TranslatedContent
	}
	return q.RawContent
}

// lcQuestionDetailResp is the payload of the questionData GraphQL query.
type lcQuestionDetailResp struct {
	Data struct {
		Question lcQuestion `json:"question"`
	} `json:"data"`
}
