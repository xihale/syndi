package routes

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var steamSpecialsRoute = routeutils.RouteSpec{
	Path:        "specials",
	Name:        "Steam Specials",
	Example:     "steam/specials",
	Maintainers: []string{"xihale"},
	Description: "Current Steam special offers (discounted games) from the search page",
	Categories:  []models.Category{{Name: "game"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Maximum number of deals (default 25, max 100)"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  SteamSpecialsHandler,
}

// SteamSpecialsHandler handles /steam/specials
//
// The storesearch JSON API carries no discount information and the
// /search/results/?specials=1&json=1 endpoint returns items without links or
// prices, so we scrape the regular specials search page instead.
func SteamSpecialsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 25, 100)

	ctx, cancel := context.WithTimeout(c.Parent(), 30*time.Second)
	defer cancel()

	url := "https://store.steampowered.com/search/?specials=1"

	doc, err := routeutils.GetHTML(ctx, c.Client(), url)
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"Steam Specials",
		url,
		"Current special offers on the Steam store",
	)

	doc.Each("a.search_result_row", func(i int, row *parser.Selection) {
		if i >= limit {
			return
		}
		routeutils.AddItem(feed, parseSteamSpecialItem(row))
	})

	return feed, nil
}

func parseSteamSpecialItem(row *parser.Selection) *models.Item {
	if row == nil {
		return nil
	}
	link := strings.TrimSpace(row.AttrOr("href", ""))
	title := strings.TrimSpace(row.Find("span.title").First().Text())
	if link == "" || title == "" {
		return nil
	}

	var b strings.Builder
	if capsule := row.Find("div.search_capsule img").First(); capsule.Size() > 0 {
		if src := capsule.AttrOr("src", ""); src != "" {
			fmt.Fprintf(&b, `<img src="%s"/><br/>`, html.EscapeString(src))
		}
	}
	if pct := strings.TrimSpace(row.Find(".discount_pct").First().Text()); pct != "" {
		b.WriteString("<strong>" + html.EscapeString(pct) + "</strong> ")
	}
	if orig := strings.TrimSpace(row.Find(".discount_original_price").First().Text()); orig != "" {
		b.WriteString(`<span style="text-decoration:line-through;">` + html.EscapeString(orig) + "</span> ")
	}
	if final := strings.TrimSpace(row.Find(".discount_final_price").First().Text()); final != "" {
		b.WriteString("<strong>" + html.EscapeString(final) + "</strong>")
	}
	if released := strings.TrimSpace(row.Find(".search_released").First().Text()); released != "" {
		b.WriteString("<br/>Released: " + html.EscapeString(released))
	}

	item := routeutils.NewItem(title, link, b.String(), time.Time{})
	item.GUID = link

	var categories []string
	if platforms := row.Find(".search_platforms span.platform_img"); platforms.Size() > 0 {
		platforms.Each(func(_ int, p *parser.Selection) {
			if cls := p.AttrOr("class", ""); cls != "" {
				categories = append(categories, strings.TrimSpace(strings.TrimPrefix(cls, "platform_img ")))
			}
		})
	}
	if review := row.Find("span.search_review_summary").First(); review.Size() > 0 {
		if tip := review.AttrOr("data-tooltip-html", ""); tip != "" {
			if idx := strings.Index(tip, "<br>"); idx > 0 {
				categories = append(categories, strings.TrimSpace(tip[:idx]))
			}
		}
	}
	routeutils.SetCategories(item, categories...)

	return item
}
