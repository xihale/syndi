package routes

import (
	"fmt"
	"html"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	ctxpkg "github.com/xihale/syndi/pkg/context"
)

// Public Twitch web GraphQL client id (the same constant used by streamlink
// and RSSHub; it is not a secret).
const twitchClientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"

const twitchGQLURL = "https://gql.twitch.tv/gql"

var twitchGQL = disguise.Chrome().
	WithHeader("Client-ID", twitchClientID).
	JSONAccept()

// gqlCall is a single GraphQL operation sent to the Twitch GQL endpoint.
type gqlCall struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// twitchUser is the shared user projection returned by every query below.
type twitchUser struct {
	ID              string          `json:"id"`
	Login           string          `json:"login"`
	DisplayName     string          `json:"displayName"`
	Description     string          `json:"description"`
	ProfileImageURL string          `json:"profileImageURL"`
	Stream          *twitchStream   `json:"stream"`
	Videos          twitchVideoEdge `json:"videos"`
	Channel         *twitchChannel  `json:"channel"`
}

type twitchStream struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	CreatedAt    time.Time      `json:"createdAt"`
	ViewersCount int64          `json:"viewersCount"`
	Game         *twitchGameRef `json:"game"`
}

type twitchGameRef struct {
	DisplayName string `json:"displayName"`
}

// Schedule categories come back as { name }, unlike game refs ({ displayName }).
type twitchCategoryRef struct {
	Name string `json:"name"`
}

type twitchVideoEdge struct {
	Edges []struct {
		Node struct {
			ID            string         `json:"id"`
			Title         string         `json:"title"`
			PublishedAt   time.Time      `json:"publishedAt"`
			LengthSeconds int64          `json:"lengthSeconds"`
			Preview       string         `json:"previewThumbnailURL"`
			Game          *twitchGameRef `json:"game"`
		} `json:"node"`
	} `json:"edges"`
}

type twitchChannel struct {
	Schedule *struct {
		Segments []twitchScheduleSegment `json:"segments"`
	} `json:"schedule"`
}

type twitchScheduleSegment struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	StartAt    time.Time           `json:"startAt"`
	EndAt      time.Time           `json:"endAt"`
	Categories []twitchCategoryRef `json:"categories"`
}

type twitchEnvelope struct {
	Data struct {
		User *twitchUser `json:"user"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// twitchQuery runs a handwritten GQL operation against the public endpoint.
// The persisted-query hashes used by upstream RSSHub have rotted away, so all
// three routes use explicit query documents instead.
func twitchQuery(c *ctxpkg.Context, call gqlCall) (*twitchUser, error) {
	var envelopes []twitchEnvelope
	if err := twitchGQL.PostJSON(twitchGQLURL, []gqlCall{call}).GetJSON(c.Parent(), c.Client(), &envelopes); err != nil {
		return nil, err
	}
	if len(envelopes) == 0 {
		return nil, fmt.Errorf("twitch: empty GraphQL response")
	}
	if len(envelopes[0].Errors) > 0 {
		return nil, fmt.Errorf("twitch: %s", envelopes[0].Errors[0].Message)
	}
	return envelopes[0].Data.User, nil
}

func twitchRequireUser(u *twitchUser, login string) (*twitchUser, error) {
	if u == nil || u.ID == "" {
		return nil, fmt.Errorf("Twitch channel %q does not exist", login)
	}
	return u, nil
}

func twitchLoginLink(login string) string { return "https://www.twitch.tv/" + login }

func escapeHTMLText(s string) string { return html.EscapeString(s) }

var _ = time.Time{} // keep time import when helpers change
