package routes

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var vscodeExtensionRoute = routeutils.RouteSpec{
	Path:        "extension/:publisher/:name",
	Name:        "VS Code Extension Versions",
	Example:     "vscode-marketplace/extension/esbenp/prettier-vscode",
	Maintainers: []string{"xihale"},
	Description: "Fetch version history of an extension from Visual Studio Marketplace",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("publisher", "Extension publisher"),
		routeutils.RequiredParam("name", "Extension name"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  VSCodeExtensionHandler,
}

// VSCodeExtensionHandler handles /vscode-marketplace/extension/:publisher/:name
func VSCodeExtensionHandler(c *ctxpkg.Context) (*models.Feed, error) {
	publisher := c.Param("publisher")
	name := c.Param("name")
	fullName := publisher + "." + name
	ctx := c.Parent()

	payload := fmt.Sprintf(
		`{"filters":[{"criteria":[{"filterType":7,"value":%q}],"pageNumber":1,"pageSize":1,"sortBy":0,"sortOrder":0}],"assetTypes":[],"flags":438}`,
		fullName,
	)
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json;api-version=3.0-preview.1",
	}
	body, err := c.Client().PostWithHeaders(ctx, "https://marketplace.visualstudio.com/_apis/public/gallery/extensionquery", []byte(payload), headers)
	if err != nil {
		return nil, err
	}

	var response VSCodeQueryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode marketplace response: %w", err)
	}
	if len(response.Results) == 0 || len(response.Results[0].Extensions) == 0 {
		return nil, fmt.Errorf("extension %s not found on Visual Studio Marketplace", fullName)
	}

	displayName := strings.TrimSpace(response.Results[0].Extensions[0].DisplayName)
	if displayName == "" {
		displayName = fullName
	}

	pageURL := fmt.Sprintf("https://marketplace.visualstudio.com/items?itemName=%s", fullName)
	feed := routeutils.NewFeed(
		fmt.Sprintf("%s - Visual Studio Marketplace", displayName),
		pageURL,
		fmt.Sprintf("Version history of %s", fullName),
	)
	routeutils.AppendMappedItems(feed, response.Results[0].Extensions[0].Versions, 30, func(version VSCodeVersion) *models.Item {
		if version.Version == "" {
			return nil
		}

		item := routeutils.NewItem(
			fmt.Sprintf("%s v%s", fullName, version.Version),
			pageURL,
			"",
			version.ParsedTime(),
		)
		item.GUID = fmt.Sprintf("vscode-%s-%s", fullName, version.Version)
		routeutils.SetCategories(item, version.Version)
		return item
	})

	return feed, nil
}

type VSCodeQueryResponse struct {
	Results []struct {
		Extensions []struct {
			Extension   string          `json:"extensionName"`
			DisplayName string          `json:"displayName"`
			Versions    []VSCodeVersion `json:"versions"`
		} `json:"extensions"`
	} `json:"results"`
}

type VSCodeVersion struct {
	Version     string `json:"version"`
	LastUpdated string `json:"lastUpdated"`
}

func (v VSCodeVersion) ParsedTime() time.Time {
	if v.LastUpdated == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v.LastUpdated)
	if err != nil {
		return time.Time{}
	}
	return t
}
