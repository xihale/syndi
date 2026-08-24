// Package routes implements RSSHub-style routes for Wallhaven.
package routes

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var wallhavenSearchRoute = routeutils.RouteSpec{
	Path:        "search",
	Name:        "Wallhaven Search",
	Example:     "wallhaven/search?q=nature",
	Maintainers: []string{"xihale"},
	Description: "Latest hot wallpapers from Wallhaven (SFW general category)",
	Categories:  []models.Category{{Name: "picture"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("q", "Search query"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  WallhavenSearchHandler,
}

type wallhavenResponse struct {
	Data []wallhavenWallpaper `json:"data"`
}

type wallhavenWallpaper struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Path   string `json:"path"`
	Thumbs struct {
		Small string `json:"small"`
		Large string `json:"large"`
	} `json:"thumbs"`
	Purity     string `json:"purity"`
	Category   string `json:"category"`
	Resolution string `json:"resolution"`
	CreatedAt  string `json:"created_at"` // "2026-08-22 10:07:10"
}

// WallhavenSearchHandler handles /wallhaven/search
func WallhavenSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	q := strings.TrimSpace(c.QueryParam("q"))

	ctx, cancel := context.WithTimeout(c.Parent(), 30*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf(
		"https://wallhaven.cc/api/v1/search?q=%s&categories=100&purity=100&sorting=hot",
		url.QueryEscape(q),
	)

	var resp wallhavenResponse
	if err := routeutils.GetJSON(ctx, c.Client(), apiURL, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"Wallhaven Hot Wallpapers",
		"https://wallhaven.cc/",
		"Currently hot wallpapers on Wallhaven",
	)
	for _, w := range resp.Data {
		routeutils.AddItem(feed, newWallhavenItem(w))
	}

	return feed, nil
}

func newWallhavenItem(w wallhavenWallpaper) *models.Item {
	if w.ID == "" || w.URL == "" {
		return nil
	}
	title := "Wallpaper " + w.ID
	if w.Resolution != "" {
		title += " (" + w.Resolution + ")"
	}
	image := w.Thumbs.Small
	if image == "" {
		image = w.Thumbs.Large
	}

	var b strings.Builder
	if image != "" {
		fmt.Fprintf(&b, `<img src="%s"/>`, html.EscapeString(image))
	}

	item := routeutils.NewItem(title, w.URL, b.String(), parseWallhavenDate(w.CreatedAt))
	item.GUID = "wallhaven-" + w.ID
	routeutils.SetCategories(item, w.Category, w.Purity)
	return item
}

// parseWallhavenDate parses the API's "2006-01-02 15:04:05" timestamps.
func parseWallhavenDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
	if err != nil {
		return time.Time{}
	}
	return t
}
