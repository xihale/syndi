package routes

import (
	"fmt"
	"time"

	ctxpkg "github.com/rsshub/go/pkg/context"
	"github.com/rsshub/go/pkg/models"
	"github.com/rsshub/go/pkg/registry"
	"github.com/rsshub/go/internal/routeutils"
)

func init() {
	route := &models.Route{
		Path:         "/npm/:package",
		Name:         "npm Package Versions",
		Example:      "npm/react",
		Maintainers:  []string{"yourname"},
		Description:  "Fetch versions from an npm package",
		Categories:   []models.Category{{Name: "programming"}},
		Features:     models.Features{SupportRadar: true},
		Handler:      NPMPackageHandler,
		Parameters: []models.Parameter{
			{Name: "package", Required: true, Description: "npm package name"},
		},
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// NPMPackageHandler handles /npm/:package
func NPMPackageHandler(c *ctxpkg.Context) (*models.Feed, error) {
	packageName := c.Param("package")
	ctx := c.Parent()

	url := fmt.Sprintf("https://registry.npmjs.org/%s", packageName)

	var response NPMResponse
	if err := routeutils.GetJSON(ctx, c.Client(), url, &response); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s - npm", packageName),
		fmt.Sprintf("https://www.npmjs.com/package/%s", packageName),
		fmt.Sprintf("Versions of npm package %s", packageName),
	)

	// Set author if available
	if response.Maintainers != nil && len(response.Maintainers) > 0 {
		routeutils.SetFeedAuthor(feed, response.Maintainers[0].Name)
	}

	// Get versions sorted by time
	items := make([]NPMDistTag, 0, len(response.Versions))
	for versionStr := range response.Versions {
		items = append(items, NPMDistTag{
			Version: versionStr,
			Time:    response.Time[versionStr],
		})
	}

	// Sort by time (newest first) and limit to 20
	for i := len(items) - 1; i >= 0 && len(feed.Items) < 20; i-- {
		v := items[i]
		if v.Time.IsZero() {
			continue
		}

		versionURL := fmt.Sprintf("https://www.npmjs.com/package/%s/v/%s", packageName, v.Version)

		description := fmt.Sprintf("Version %s of %s", v.Version, packageName)
		if response.Homepage != "" {
			description += fmt.Sprintf("<br/><a href=\"%s\">Homepage</a>", response.Homepage)
		}
		if response.Repository.URL != "" {
			description += fmt.Sprintf("<br/><a href=\"%s\">Repository</a>", formatRepoURL(response.Repository.URL))
		}

		item := routeutils.NewItem(
			fmt.Sprintf("%s@%s", packageName, v.Version),
			versionURL,
			description,
			v.Time,
		)
		item.GUID = fmt.Sprintf("npm-%s-%s", packageName, v.Version)

		routeutils.SetCategories(item, "version", v.Version)

		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

// formatRepoURL converts git+https URLs to normal https URLs
func formatRepoURL(url string) string {
	if len(url) > 11 && url[:11] == "git+https://" {
		return "https://" + url[11:]
	}
	if len(url) > 10 && url[:10] == "git+ssh://" {
		return url
	}
	return url
}

type NPMResponse struct {
	ID          string                   `json:"_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Homepage    string                   `json:"homepage"`
	Keywords    []string                 `json:"keywords"`
	Version     string                   `json:"dist-tags.latest"`
	Versions    map[string]NPMVersion    `json:"versions"`
	Time        map[string]time.Time     `json:"time"`
	Maintainers []NPMMaintainer          `json:"maintainers"`
	Repository  NPMRepository            `json:"repository"`
}

type NPMVersion struct {
	ID       string `json:"_id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Dist     NPMDist `json:"dist"`
}

type NPMDist struct {
	Tarball string `json:"tarball"`
	Shasum  string `json:"shasum"`
}

type NPMRepository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type NPMMaintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type NPMDistTag struct {
	Version string
	Time    time.Time
}
