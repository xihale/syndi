package routes

import (
	"fmt"
	"html"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var amoAddonRoute = routeutils.RouteSpec{
	Path:        "addon/:slug",
	Name:        "Firefox Add-on Versions",
	Example:     "amo/addon/ublock-origin",
	Maintainers: []string{"xihale"},
	Description: "Fetch latest version releases of a Firefox add-on from addons.mozilla.org",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("slug", "Add-on slug, e.g. ublock-origin"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  AMOAddonHandler,
}

// AMOAddonHandler handles /amo/addon/:slug
func AMOAddonHandler(c *ctxpkg.Context) (*models.Feed, error) {
	slug := c.Param("slug")
	ctx := c.Parent()

	url := fmt.Sprintf("https://addons.mozilla.org/api/v4/addons/addon/%s/versions/?page_size=25", slug)
	headers := map[string]string{"User-Agent": "Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0"}

	var response AMOVersionsResponse
	if err := routeutils.GetJSONWithHeaders(ctx, c.Client(), url, headers, &response); err != nil {
		return nil, err
	}

	addonURL := fmt.Sprintf("https://addons.mozilla.org/firefox/addon/%s/", slug)
	feed := routeutils.NewFeed(
		fmt.Sprintf("%s Versions - Firefox Add-ons", slug),
		addonURL,
		fmt.Sprintf("Latest versions of Firefox add-on %s", slug),
	)
	routeutils.AppendMappedItems(feed, response.Results, 25, func(version AMOVersion) *models.Item {
		title := fmt.Sprintf("%s version %s", slug, version.Version)

		description := ""
		download := ""
		if len(version.Files) > 0 {
			download = fmt.Sprintf("<br/><a href=\"%s\">Download</a>", version.Files[0].URL)
		}
		notes := amoReleaseNotes(version.ReleaseNotes)
		if notes != "" {
			description = html.EscapeString(notes)
		}

		item := routeutils.NewItem(title, addonURL, description+download, version.ParsedTime())
		item.GUID = fmt.Sprintf("amo-%s-%s", slug, version.Version)
		routeutils.SetCategories(item, version.Version)
		return item
	})

	return feed, nil
}

// amoReleaseNotes extracts release notes; the API may localize them as a map.
func amoReleaseNotes(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return v
	case map[string]interface{}:
		for _, locale := range []string{"en-US", "en"} {
			if s, ok := v[locale].(string); ok && s != "" {
				return s
			}
		}
		for _, val := range v {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

type AMOVersionsResponse struct {
	Results []AMOVersion `json:"results"`
}

type AMOVersion struct {
	Version      string      `json:"version"`
	ReleaseNotes interface{} `json:"release_notes"`
	Files        []struct {
		Created string `json:"created"`
		URL     string `json:"url"`
	} `json:"files"`
}

func (v AMOVersion) ParsedTime() time.Time {
	if len(v.Files) == 0 || v.Files[0].Created == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v.Files[0].Created)
	if err != nil {
		return time.Time{}
	}
	return t
}
