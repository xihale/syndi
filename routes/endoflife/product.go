package routes

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

const endOfLifeBaseURL = "https://endoflife.date/"

var endOfLifeProductRoute = routeutils.RouteSpec{
	Path:        "product/:product",
	Name:        "End of Life Product Releases",
	Example:     "endoflife/product/nodejs",
	Maintainers: []string{"xihale"},
	Description: "Release cycles and end-of-life dates for a product tracked by endoflife.date",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("product", "product slug used by endoflife.date (e.g. nodejs)"),
	},
	CacheTTL: 12 * time.Hour,
	Handler:  EndOfLifeProductHandler,
}

// EndOfLifeProductHandler handles /endoflife/product/:product
func EndOfLifeProductHandler(c *ctxpkg.Context) (*models.Feed, error) {
	product := strings.TrimSpace(c.Param("product"))
	if product == "" || !isValidProductSlug(product) {
		return nil, fmt.Errorf("invalid product %q", product)
	}

	url := endOfLifeBaseURL + "api/" + url.PathEscape(product) + ".json"
	var cycles []eolCycle
	if err := routeutils.GetJSON(c.Parent(), c.Client(), url, &cycles); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s releases and EOL dates", product),
		endOfLifeBaseURL+product,
		fmt.Sprintf("Release cycles and end-of-life dates for %s, tracked by endoflife.date", product),
	)

	routeutils.AppendMappedItems(feed, cycles, 0, func(cy eolCycle) *models.Item {
		if cy.Cycle == "" && cy.Latest == "" {
			return nil
		}
		title := fmt.Sprintf("%s %s → latest %s", product, cy.Cycle, cy.Latest)
		desc := "<b>Latest:</b> " + html.EscapeString(cy.Latest)

		rows := []struct{ label, value string }{
			{"Latest release", cy.LatestReleaseDate},
			{"Released", cy.ReleaseDate},
			{"Support until", cy.Support.String()},
			{"EOL", cy.EOL.String()},
			{"LTS", cy.LTS.String()},
			{"Extended support", cy.ExtendedSupport.String()},
		}
		for _, row := range rows {
			if row.value == "" {
				continue
			}
			desc += fmt.Sprintf("<br/><b>%s:</b> %s", row.label, html.EscapeString(row.value))
		}

		pubDate := time.Time{}
		if parsed, err := dateutil.ParseDate(cy.LatestReleaseDate); err == nil {
			pubDate = parsed
		}
		item := routeutils.NewItem(title, endOfLifeBaseURL+product, desc, pubDate)
		item.GUID = "endoflife-" + product + "-" + cy.Cycle
		return item
	})
	return feed, nil
}

// isValidProductSlug restricts the upstream path segment to safe characters.
func isValidProductSlug(product string) bool {
	for _, r := range product {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

// eolFlexible holds a JSON value that may be a date string, a boolean, or null.
type eolFlexible string

// UnmarshalJSON accepts strings, booleans and null for flexible fields.
func (v *eolFlexible) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	switch s {
	case "", "null":
		*v = ""
	case "true", "false":
		*v = eolFlexible(s)
	default:
		*v = eolFlexible(strings.Trim(s, `"`))
	}
	return nil
}

// String renders the flexible value for display; booleans become Yes/No.
func (v eolFlexible) String() string {
	switch v {
	case "":
		return ""
	case "true":
		return "Yes"
	case "false":
		return "No"
	default:
		return string(v)
	}
}

type eolCycle struct {
	Cycle             string      `json:"cycle"`
	Latest            string      `json:"latest"`
	LatestReleaseDate string      `json:"latestReleaseDate"`
	ReleaseDate       string      `json:"releaseDate"`
	EOL               eolFlexible `json:"eol"`
	Support           eolFlexible `json:"support"`
	LTS               eolFlexible `json:"lts"`
	ExtendedSupport   eolFlexible `json:"extendedSupport"`
}
