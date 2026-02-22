package routes

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
)

func init() {
	cacheTTL := 4 * time.Hour // npm package versions change infrequently

	route := &models.Route{
		Path:        "/npm/:package",
		Name:        "npm Package Versions",
		Example:     "npm/react",
		Maintainers: []string{"yourname"},
		Description: "Fetch versions from an npm package",
		Categories:  []models.Category{{Name: "programming"}},
		Features:    models.Features{SupportRadar: true},
		Handler:     NPMPackageHandler,
		Parameters: []models.Parameter{
			{Name: "package", Required: true, Description: "npm package name"},
		},
		CacheTTL: &cacheTTL,
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
	if len(response.Maintainers) > 0 {
		routeutils.SetFeedAuthor(feed, response.Maintainers[0].Name)
	}

	// Sort versions by publish time (newest first) and emit the first 20.
	for _, v := range sortedVersionTags(response.Versions, response.Time) {
		if v.Time.IsZero() {
			continue
		}
		if len(feed.Items) >= 20 {
			break
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
func formatRepoURL(repoURL string) string {
	repoURL = strings.TrimSpace(repoURL)
	repoURL = strings.TrimPrefix(repoURL, "git+")

	if strings.HasPrefix(repoURL, "git://github.com/") {
		repoURL = "https://github.com/" + strings.TrimPrefix(repoURL, "git://github.com/")
	}
	if strings.HasPrefix(repoURL, "ssh://git@github.com/") {
		repoURL = "https://github.com/" + strings.TrimPrefix(repoURL, "ssh://git@github.com/")
	}
	if strings.HasPrefix(repoURL, "git@github.com:") {
		repoURL = "https://github.com/" + strings.TrimPrefix(repoURL, "git@github.com:")
	}

	if strings.HasPrefix(repoURL, "https://github.com/") && strings.HasSuffix(repoURL, ".git") {
		repoURL = strings.TrimSuffix(repoURL, ".git")
	}
	return repoURL
}

func sortedVersionTags(versions map[string]NPMVersion, versionTimes map[string]time.Time) []NPMDistTag {
	items := make([]NPMDistTag, 0, len(versions))
	for version := range versions {
		publishedAt := versionTimes[version]
		items = append(items, NPMDistTag{
			Version: version,
			Time:    publishedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.Time.Equal(right.Time) {
			return left.Version > right.Version
		}
		return left.Time.After(right.Time)
	})

	return items
}

type NPMResponse struct {
	ID          string                `json:"_id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Homepage    string                `json:"homepage"`
	Keywords    []string              `json:"keywords"`
	Version     string                `json:"dist-tags.latest"`
	Versions    map[string]NPMVersion `json:"versions"`
	Time        map[string]time.Time  `json:"time"`
	Maintainers []NPMMaintainer       `json:"maintainers"`
	Repository  NPMRepository         `json:"repository"`
}

type NPMVersion struct {
	ID      string  `json:"_id"`
	Name    string  `json:"name"`
	Version string  `json:"version"`
	Dist    NPMDist `json:"dist"`
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
