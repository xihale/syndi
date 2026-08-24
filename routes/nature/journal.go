package routes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var natureJournalSlug = regexp.MustCompile(`^[a-z]+$`)

var natureJournalRoute = routeutils.RouteSpec{
	Path:        "journal/:journal",
	Name:        "Nature Journal",
	Example:     "nature/journal/nature",
	Maintainers: []string{"xihale"},
	Description: "Latest research from a Nature Portfolio journal (e.g. nature, genetics, physics)",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("journal", "journal slug, e.g. nature"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  NatureJournalHandler,
}

// NatureJournalHandler handles /nature/journal/:journal
func NatureJournalHandler(c *ctxpkg.Context) (*models.Feed, error) {
	journal := strings.TrimSpace(c.Param("journal"))
	if !natureJournalSlug.MatchString(journal) {
		return nil, fmt.Errorf("invalid journal slug %q (lowercase letters only)", journal)
	}

	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), fmt.Sprintf("https://www.nature.com/%s.rss", journal))
	if err != nil {
		return nil, err
	}
	feed.Title = fmt.Sprintf("Nature: %s", journal)
	feed.Link = fmt.Sprintf("https://www.nature.com/%s/", journal)
	feed.Description = fmt.Sprintf("Latest research from the %s journal", journal)
	return feed, nil
}
