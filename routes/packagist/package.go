package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var packagistPackageRoute = routeutils.RouteSpec{
	Path:        "package/:vendor/:name",
	Name:        "Packagist Package Versions",
	Example:     "packagist/package/symfony/console",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest version releases from a Packagist package",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("vendor", "Package vendor"),
		routeutils.RequiredParam("name", "Package name"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  PackagistPackageHandler,
}

// PackagistPackageHandler handles /packagist/package/:vendor/:name
func PackagistPackageHandler(c *ctxpkg.Context) (*models.Feed, error) {
	vendor := c.Param("vendor")
	name := c.Param("name")
	fullName := vendor + "/" + name
	ctx := c.Parent()

	url := fmt.Sprintf("https://repo.packagist.org/p2/%s.json", fullName)

	var response PackagistP2Response
	if err := routeutils.GetJSON(ctx, c.Client(), url, &response); err != nil {
		return nil, err
	}

	versions, ok := response.Packages[fullName]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("no releases found for %s", fullName)
	}

	// p2 metadata may be minified: omitted fields repeat the previous entry's
	// value and "__unset" marks an explicitly removed value.
	description := ""
	if v := versions[0].Description(); v != "" && v != "__unset" {
		description = v
	}
	feed := routeutils.NewFeed(
		fmt.Sprintf("%s - Packagist", fullName),
		fmt.Sprintf("https://packagist.org/packages/%s", fullName),
		html.EscapeString(description),
	)

	count := 0
	for _, version := range versions {
		if count >= 30 {
			break
		}
		if v := version.Description(); v != "" && v != "__unset" {
			description = v
		} else if v == "__unset" {
			description = ""
		}
		published := version.ParsedTime()
		if published.IsZero() {
			continue
		}

		link := fmt.Sprintf("https://packagist.org/packages/%s#%s", fullName, version.Version)
		item := routeutils.NewItem(
			fmt.Sprintf("%s %s", fullName, version.Version),
			link,
			html.EscapeString(description),
			published,
		)
		item.GUID = link
		routeutils.AddItem(feed, item)
		count++
	}

	return feed, nil
}

type PackagistP2Response struct {
	Packages map[string][]PackagistVersion `json:"packages"`
}

type PackagistVersion struct {
	Version        string      `json:"version"`
	Time           string      `json:"time"`
	PublishedTime  string      `json:"published-time"`
	RawDescription interface{} `json:"description"`
}

func (v PackagistVersion) Description() string {
	switch d := v.RawDescription.(type) {
	case string:
		return strings.TrimSpace(d)
	case nil:
		return ""
	default:
		return ""
	}
}

func (v PackagistVersion) ParsedTime() time.Time {
	raw := v.Time
	if raw == "" {
		raw = v.PublishedTime
	}
	if raw == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t
	}
	return time.Time{}
}
