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

var plosJournalSlug = regexp.MustCompile(`^[a-z]+$`)

var plosJournalRoute = routeutils.RouteSpec{
	Path:        "journal/:journal",
	Name:        "PLOS Journal",
	Example:     "plos/journal/plosone",
	Maintainers: []string{"xihale"},
	Description: "Latest articles from a PLOS journal (e.g. one/plosone, biology/plosbiology)",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("journal", "journal slug without the plos prefix, e.g. one"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  PLOSJournalHandler,
}

// PLOSJournalHandler handles /plos/journal/:journal
func PLOSJournalHandler(c *ctxpkg.Context) (*models.Feed, error) {
	journal := strings.TrimSpace(c.Param("journal"))
	if !plosJournalSlug.MatchString(journal) {
		return nil, fmt.Errorf("invalid journal slug %q (lowercase letters only)", journal)
	}
	// Accept both "one" and "plosone" spellings; upstream path always needs the plos prefix.
	journal = strings.TrimPrefix(journal, "plos")

	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), fmt.Sprintf("https://journals.plos.org/plos%s/feed/atom", journal))
	if err != nil {
		return nil, err
	}
	feed.Title = fmt.Sprintf("PLOS %s", strings.TrimPrefix(strings.ToLower(journal), "plos"))
	feed.Link = fmt.Sprintf("https://journals.plos.org/plos%s/", journal)
	feed.Description = fmt.Sprintf("Latest articles from PLOS %s", journal)
	return feed, nil
}
