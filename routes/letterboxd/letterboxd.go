package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// letterboxdProfile keeps requests browser-like; Letterboxd tolerates plain
// clients today but the poster API is CDN-fronted and picky about Referer,
// which the profile supplies automatically.
var letterboxdProfile = disguise.Chrome()

const letterboxdBase = "https://letterboxd.com"

type letterboxdPoster struct {
	URL   string `json:"url"`
	URL2X string `json:"url2x"`
}

var letterboxdWatchlistRoute = routeutils.RouteSpec{
	Path:        "watchlist/:username",
	Name:        "User Watchlist",
	Example:     "letterboxd/watchlist/matthew",
	Maintainers: []string{"xihale"},
	Description: "Films on a Letterboxd user's watchlist",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("username", "Letterboxd username"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  LetterboxdWatchlistHandler,
}

// LetterboxdWatchlistHandler handles /letterboxd/watchlist/:username.
func LetterboxdWatchlistHandler(c *ctxpkg.Context) (*models.Feed, error) {
	username := strings.TrimPrefix(c.Param("username"), "/")
	pageURL := fmt.Sprintf("%s/%s/watchlist/", letterboxdBase, username)

	doc, err := letterboxdProfile.Fetch(pageURL).GetHTML(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())
	if title == "" {
		title = username + "'s Watchlist • Letterboxd"
	}
	feed := routeutils.NewFeed(title, pageURL, "Films "+username+" wants to watch on Letterboxd")

	posters := doc.Find(`div.react-component[data-component-class*="LazyPoster"]`)
	if posters.Length() == 0 {
		return nil, fmt.Errorf("no watchlist entries found for %q (profile may be private or empty)", username)
	}
	posters.Each(func(_ int, s *goquery.Selection) {
		linkPath := firstNonEmpty(s.AttrOr("data-item-link", ""), s.AttrOr("data-target-link", ""))
		if linkPath == "" {
			return
		}
		link := letterboxdBase + "/" + strings.TrimLeft(linkPath, "/")

		name := firstNonEmpty(
			s.AttrOr("data-item-full-display-name", ""),
			s.AttrOr("data-item-name", ""),
		)
		if name == "" {
			return
		}

		desc := html.EscapeString(name)
		var poster letterboxdPoster
		if err := routeutils.GetJSON(c.Parent(), c.Client(), filmPosterURL(link), &poster); err == nil {
			img := firstNonEmpty(poster.URL2X, poster.URL)
			if img != "" {
				desc = fmt.Sprintf(`<img src="%s" alt=""><br>`, html.EscapeString(img)) + desc
			}
		}

		item := routeutils.NewItem(name, link, desc, time.Time{})
		routeutils.AddItem(feed, item)
	})
	return feed, nil
}

func filmPosterURL(filmLink string) string {
	return fmt.Sprintf("%sposter/std/125/", filmLink)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
