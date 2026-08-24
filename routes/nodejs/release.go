package routes

import (
	"html"
	"sort"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

const nodejsReleaseLimit = 20

var nodejsReleaseRoute = routeutils.RouteSpec{
	Path:        "release",
	Name:        "Node.js Releases",
	Example:     "nodejs/release",
	Maintainers: []string{"xihale"},
	Description: "Latest Node.js releases from the official dist index",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     NodejsReleaseHandler,
}

// nodejsLTS accepts both JSON booleans and LTS codename strings
// (the dist index uses false for non-LTS lines, e.g. "Krypton" otherwise).
type nodejsLTS bool

func (b *nodejsLTS) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	*b = s != "" && s != "false" && s != "null"
	return nil
}

type nodejsRelease struct {
	Version  string    `json:"version"`
	Date     string    `json:"date"`
	NPM      string    `json:"npm"`
	V8       string    `json:"v8"`
	LTS      nodejsLTS `json:"lts"`
	Security bool      `json:"security"`
}

// NodejsReleaseHandler handles /nodejs/release
func NodejsReleaseHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var releases []nodejsRelease
	if err := routeutils.GetJSON(c.Parent(), c.Client(), "https://nodejs.org/dist/index.json", &releases); err != nil {
		return nil, err
	}

	parsed := make([]nodejsRelease, 0, len(releases))
	dates := make(map[string]time.Time, len(releases))
	for _, r := range releases {
		if r.Version == "" {
			continue
		}
		t, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			continue
		}
		parsed = append(parsed, r)
		dates[r.Date] = t
	}
	sort.SliceStable(parsed, func(i, j int) bool {
		return dates[parsed[i].Date].After(dates[parsed[j].Date])
	})
	if len(parsed) > nodejsReleaseLimit {
		parsed = parsed[:nodejsReleaseLimit]
	}

	feed := routeutils.NewFeed(
		"Node.js Releases",
		"https://nodejs.org/en/blog/release",
		"The latest Node.js releases",
	)
	for _, r := range parsed {
		var desc strings.Builder
		desc.WriteString("npm " + html.EscapeString(r.NPM) + ", V8 " + html.EscapeString(r.V8))
		if r.LTS {
			desc.WriteString(", LTS")
		}
		if r.Security {
			desc.WriteString(", security release")
		}
		item := routeutils.NewItem(
			r.Version,
			"https://nodejs.org/en/blog/release/"+strings.TrimPrefix(r.Version, "v"),
			desc.String(),
			dates[r.Date],
		)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
