package routes

import (
	"fmt"
	"html"
	"sort"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var pypiPackageRoute = routeutils.RouteSpec{
	Path:        "package/:package",
	Name:        "PyPI Package Versions",
	Example:     "pypi/package/requests",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest version releases from a PyPI package",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("package", "PyPI package name"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  PyPIPackageHandler,
}

// PyPIPackageHandler handles /pypi/package/:package
func PyPIPackageHandler(c *ctxpkg.Context) (*models.Feed, error) {
	pkg := c.Param("package")
	ctx := c.Parent()

	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", pkg)

	var response PyPIResponse
	if err := routeutils.GetJSON(ctx, c.Client(), url, &response); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s - PyPI", pkg),
		fmt.Sprintf("https://pypi.org/project/%s/", pkg),
		html.EscapeString(response.Info.Summary),
	)
	routeutils.AppendMappedItems(feed, sortedPyPIReleases(response.Releases), 20, func(rel pypiRelease) *models.Item {
		link := fmt.Sprintf("https://pypi.org/project/%s/%s/", pkg, rel.Version)
		item := routeutils.NewItem(
			fmt.Sprintf("%s %s", pkg, rel.Version),
			link,
			html.EscapeString(response.Info.Summary),
			rel.Time,
		)
		item.GUID = link
		return item
	})

	return feed, nil
}

type pypiRelease struct {
	Version string
	Time    time.Time
}

// sortedPyPIReleases flattens the releases map into newest-first slice.
func sortedPyPIReleases(releases map[string][]PyPIFile) []pypiRelease {
	items := make([]pypiRelease, 0, len(releases))
	for version, files := range releases {
		if len(files) == 0 {
			continue
		}
		var uploaded time.Time
		for _, f := range files {
			if t, err := time.Parse(time.RFC3339, f.UploadTime); err == nil && (uploaded.IsZero() || t.Before(uploaded)) {
				uploaded = t
			}
		}
		if uploaded.IsZero() {
			continue
		}
		items = append(items, pypiRelease{Version: version, Time: uploaded})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Time.After(items[j].Time)
	})
	return items
}

type PyPIResponse struct {
	Info struct {
		Summary string `json:"summary"`
	} `json:"info"`
	Releases map[string][]PyPIFile `json:"releases"`
}

type PyPIFile struct {
	Filename   string `json:"filename"`
	UploadTime string `json:"upload_time_iso_8601"`
}
