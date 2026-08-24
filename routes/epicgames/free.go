// Package routes implements RSSHub-style routes for the Epic Games Store.
package routes

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var epicFreeRoute = routeutils.RouteSpec{
	Path:        "free",
	Name:        "Epic Games Free Games",
	Example:     "epicgames/free",
	Maintainers: []string{"xihale"},
	Description: "Current and upcoming free games on the Epic Games Store",
	Categories:  []models.Category{{Name: "game"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     EpicGamesFreeHandler,
}

// epicFreeResponse mirrors the freeGamesPromotions payload, which serves the
// standard Catalog.searchStore element shape with price/promotions data.
type epicFreeResponse struct {
	Data struct {
		Catalog struct {
			SearchStore struct {
				Elements []epicElement `json:"elements"`
			} `json:"searchStore"`
		} `json:"Catalog"`
	} `json:"data"`
}

type epicElement struct {
	Title       string          `json:"title"`
	ID          string          `json:"id"`
	Description string          `json:"description"`
	ProductSlug string          `json:"productSlug"`
	Seller      epicSeller      `json:"seller"`
	KeyImages   []epicKeyImage  `json:"keyImages"`
	Price       *epicPrice      `json:"price"`
	Promotions  *epicPromotions `json:"promotions"`
	CatalogNs   epicCatalogNs   `json:"catalogNs"`
}

type epicSeller struct {
	Name string `json:"name"`
}

type epicCatalogNs struct {
	Mappings []struct {
		PageSlug string `json:"pageSlug"`
	} `json:"mappings"`
}

type epicKeyImage struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type epicPrice struct {
	TotalPrice struct {
		OriginalPrice int64 `json:"originalPrice"`
		FmtPrice      struct {
			OriginalPrice string `json:"originalPrice"`
			DiscountPrice string `json:"discountPrice"`
		} `json:"fmtPrice"`
	} `json:"totalPrice"`
}

type epicPromotions struct {
	PromotionalOffers         []epicPromoGroup `json:"promotionalOffers"`
	UpcomingPromotionalOffers []epicPromoGroup `json:"upcomingPromotionalOffers"`
}

type epicPromoGroup struct {
	PromotionalOffers []struct {
		StartDate time.Time `json:"startDate"`
		EndDate   time.Time `json:"endDate"`
	} `json:"promotionalOffers"`
}

type epicOfferWindow struct {
	Start time.Time
	End   time.Time
}

// EpicGamesFreeHandler handles /epicgames/free
//
// The legacy https://graphql.epicgames.com/graphql endpoint is gone (404);
// the same Catalog.searchStore promotions data is served as JSON by the
// official static storefront backend.
func EpicGamesFreeHandler(c *ctxpkg.Context) (*models.Feed, error) {
	ctx, cancel := context.WithTimeout(c.Parent(), 30*time.Second)
	defer cancel()

	url := "https://store-site-backend-static.ak.epicgames.com/freeGamesPromotions?locale=en-US&country=US&allowCountries=US"

	var resp epicFreeResponse
	if err := routeutils.GetJSON(ctx, c.Client(), url, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"Epic Games Free Games",
		"https://store.epicgames.com/en-US/free-games",
		"Current and upcoming free games on the Epic Games Store",
	)

	now := time.Now()
	for _, el := range resp.Data.Catalog.SearchStore.Elements {
		active, upcoming := epicFreeWindows(el, now)
		if len(active) == 0 && len(upcoming) == 0 {
			continue
		}
		upcomingOnly := len(active) == 0

		title := el.Title
		if upcomingOnly {
			title = "[Upcoming] " + title
		}
		link := epicGameLink(el)
		item := routeutils.NewItem(title, link, epicDescription(el), activeOrUpcomingDate(active, upcoming))
		if item == nil {
			continue
		}
		item.GUID = el.ID
		if el.Seller.Name != "" {
			routeutils.SetItemAuthor(item, el.Seller.Name, "", "")
		}
		routeutils.SetCategories(item, map[bool]string{true: "upcoming", false: "free now"}[upcomingOnly])
		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

func epicFreeWindows(el epicElement, now time.Time) (active, upcoming []epicOfferWindow) {
	if el.Promotions == nil || el.Price == nil {
		return nil, nil
	}
	// Only truly free offers: 100%-off promotions make discountPrice zero.
	free := el.Price.TotalPrice.FmtPrice.DiscountPrice == "0"
	collect := func(groups []epicPromoGroup, filter func(epicOfferWindow) bool) {
		for _, g := range groups {
			for _, o := range g.PromotionalOffers {
				w := epicOfferWindow{Start: o.StartDate, End: o.EndDate}
				if free && filter(w) {
					target := &active
					if w.Start.After(now) {
						target = &upcoming
					}
					*target = append(*target, w)
				}
			}
		}
	}
	collect(el.Promotions.PromotionalOffers, func(w epicOfferWindow) bool { return true })
	collect(el.Promotions.UpcomingPromotionalOffers, func(w epicOfferWindow) bool { return true })
	return active, upcoming
}

func epicGameLink(el epicElement) string {
	slug := strings.TrimSpace(el.ProductSlug)
	if slug == "" {
		for _, m := range el.CatalogNs.Mappings {
			if m.PageSlug != "" {
				slug = m.PageSlug
				break
			}
		}
	}
	if i := strings.Index(slug, "/"); i > 0 {
		slug = slug[:i]
	}
	if slug == "" {
		return "https://store.epicgames.com/en-US/free-games"
	}
	return "https://store.epicgames.com/en-US/p/" + slug
}

func epicImage(el epicElement, imageType string) string {
	for _, img := range el.KeyImages {
		if img.Type == imageType && img.URL != "" {
			return img.URL
		}
	}
	return ""
}

func epicDescription(el epicElement) string {
	var b strings.Builder
	if img := epicImage(el, "OfferImageWide"); img == "" {
		if img = epicImage(el, "Thumbnail"); img != "" {
			fmt.Fprintf(&b, `<img src="%s"/><br/>`, html.EscapeString(img))
		}
	} else {
		fmt.Fprintf(&b, `<img src="%s"/><br/>`, html.EscapeString(img))
	}
	if el.Description != "" {
		b.WriteString("<p>" + html.EscapeString(el.Description) + "</p>")
	}
	if el.Price != nil && el.Price.TotalPrice.FmtPrice.OriginalPrice != "" &&
		el.Price.TotalPrice.FmtPrice.OriginalPrice != "0" {
		b.WriteString("Original price: " + html.EscapeString(el.Price.TotalPrice.FmtPrice.OriginalPrice) + "<br/>")
	}
	return b.String()
}

func activeOrUpcomingDate(active, upcoming []epicOfferWindow) time.Time {
	if len(active) > 0 {
		return active[0].Start
	}
	if len(upcoming) > 0 {
		return upcoming[0].Start
	}
	return time.Time{}
}
