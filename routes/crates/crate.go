package routes

import (
	"fmt"
	"html"
	"time"

	"github.com/xihale/rsshub-go/internal/disguise"
	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var cratesCrateRoute = routeutils.RouteSpec{
	Path:        "crate/:crate",
	Name:        "crates.io Crate Versions",
	Example:     "crates/crate/serde",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest version releases from a crates.io crate",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("crate", "crate name"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  CratesCrateHandler,
}

// CratesCrateHandler handles /crates/crate/:crate
func CratesCrateHandler(c *ctxpkg.Context) (*models.Feed, error) {
	crateName := c.Param("crate")
	ctx := c.Parent()

	url := fmt.Sprintf("https://crates.io/api/v1/crates/%s/versions", crateName)

	// crates.io rejects empty User-Agent; a polite custom identity is required.
	var response CratesVersionsResponse
	if err := disguise.Custom("rsshub-go/0.1 (+https://github.com/xihale/rsshub-go)").Fetch(url).GetJSON(ctx, c.Client(), &response); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s - crates.io", crateName),
		fmt.Sprintf("https://crates.io/crates/%s", crateName),
		fmt.Sprintf("Latest versions of crate %s", crateName),
	)
	routeutils.AppendMappedItems(feed, response.Versions, 30, func(version CrateVersion) *models.Item {
		link := fmt.Sprintf("https://crates.io/crates/%s/%s", crateName, version.Num)
		description := ""
		if version.License != "" {
			description = fmt.Sprintf("License: %s", html.EscapeString(version.License))
		}
		if version.Yanked {
			if description != "" {
				description += "<br/>"
			}
			description += "<strong>yanked</strong>"
		}

		item := routeutils.NewItem(
			fmt.Sprintf("%s %s", crateName, version.Num),
			link,
			description,
			version.CreatedAt,
		)
		item.GUID = link
		routeutils.SetCategories(item, version.Num)
		if version.Yanked {
			routeutils.SetCategories(item, "yanked")
		}
		return item
	})

	return feed, nil
}

type CratesVersionsResponse struct {
	Versions []CrateVersion `json:"versions"`
}

type CrateVersion struct {
	Num       string    `json:"num"`
	CreatedAt time.Time `json:"created_at"`
	Yanked    bool      `json:"yanked"`
	License   string    `json:"license"`
}
