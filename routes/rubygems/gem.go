package routes

import (
	"fmt"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var rubyGemsGemRoute = routeutils.RouteSpec{
	Path:        "gem/:gem",
	Name:        "RubyGems Gem Versions",
	Example:     "rubygems/gem/rails",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest version releases from a RubyGems gem",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("gem", "gem name"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  RubyGemsGemHandler,
}

// RubyGemsGemHandler handles /rubygems/gem/:gem
func RubyGemsGemHandler(c *ctxpkg.Context) (*models.Feed, error) {
	gem := c.Param("gem")
	ctx := c.Parent()

	url := fmt.Sprintf("https://rubygems.org/api/v1/versions/%s.json", gem)

	var versions []RubyGemVersion
	if err := routeutils.GetJSON(ctx, c.Client(), url, &versions); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s - RubyGems", gem),
		fmt.Sprintf("https://rubygems.org/gems/%s", gem),
		fmt.Sprintf("Latest versions of gem %s", gem),
	)
	routeutils.AppendMappedItems(feed, versions, 30, func(version RubyGemVersion) *models.Item {
		link := fmt.Sprintf("https://rubygems.org/gems/%s/versions/%s", gem, version.Number)
		title := fmt.Sprintf("%s %s", gem, version.Number)
		if version.Platform != "ruby" && version.Platform != "" {
			title = fmt.Sprintf("%s (%s)", title, version.Platform)
		}

		item := routeutils.NewItem(title, link, "", version.CreatedAt)
		item.GUID = link
		if version.Platform != "" {
			routeutils.SetCategories(item, version.Platform)
		}
		if version.Prerelease {
			routeutils.SetCategories(item, "prerelease")
		}
		return item
	})

	return feed, nil
}

type RubyGemVersion struct {
	Number     string    `json:"number"`
	CreatedAt  time.Time `json:"created_at"`
	Platform   string    `json:"platform"`
	Prerelease bool      `json:"prerelease"`
}
