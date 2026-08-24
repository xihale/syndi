package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

// --- /appstore/xianmian : AppSo daily limited-free & discounted apps ---

type xianmianResp struct {
	Objects []xianmianItem `json:"objects"`
}

type xianmianItem struct {
	App struct {
		Name string `json:"name"`
		Icon struct {
			Image string `json:"image"`
		} `json:"icon"`
		DownloadLink []struct {
			Device string `json:"device"`
			Link   string `json:"link"`
			Region string `json:"region"`
			Price  string `json:"price"`
		} `json:"download_link"`
	} `json:"app"`
	Content      string `json:"content"`
	PublishedAt  int64  `json:"published_at"`
	UpdatedAt    int64  `json:"updated_at"`
	DiscountInfo []struct {
		OriginalPrice   string `json:"original_price"`
		DiscountedPrice string `json:"discounted_price"`
		Expired         bool   `json:"expired"`
	} `json:"discount_info"`
}

var appstoreXianmianRoute = routeutils.RouteSpec{
	Path:        "xianmian",
	Name:        "AppSo Daily Limited-Free",
	Example:     "appstore/xianmian",
	Maintainers: []string{"xihale"},
	Description: "Daily limited-free and discounted iOS apps curated by AppSo (鲜面连线)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     AppstoreXianmianHandler,
}

// AppstoreXianmianHandler handles /appstore/xianmian.
func AppstoreXianmianHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var resp xianmianResp
	if err := routeutils.GetJSON(c.Parent(), c.Client(),
		"https://app.so/api/v5/appso/discount/?platform=web&limit=20", &resp); err != nil {
		return nil, err
	}
	feed := routeutils.NewFeed("每日精品限免 / 促销应用", "https://app.so/xianmian/",
		"鲜面连线 by AppSo：每日精品限免 / 促销应用")
	for _, o := range resp.Objects {
		if len(o.DiscountInfo) == 0 || o.App.Name == "" || len(o.App.DownloadLink) == 0 {
			continue
		}
		d := o.DiscountInfo[0]
		kind := "降价"
		if d.DiscountedPrice == "0.00" {
			kind = "免费"
		}
		link := o.App.DownloadLink[0].Link
		desc := fmt.Sprintf(
			`<img src="%s"/><br>原价：¥%s -&gt; 现价：¥%s<br>平台：%s<br>%s`,
			html.EscapeString(o.App.Icon.Image),
			html.EscapeString(d.OriginalPrice),
			html.EscapeString(d.DiscountedPrice),
			html.EscapeString(o.App.DownloadLink[0].Device),
			o.Content, // upstream HTML-ish text
		)
		item := routeutils.NewItem(
			fmt.Sprintf("「%s」%s", kind, html.EscapeString(o.App.Name)),
			link, desc, time.Unix(o.UpdatedAt, 0))
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

// --- /appstore/price/:country/:type/:id : CheapCharts price-drop watcher ---

type cheapchartsResp struct {
	Results struct {
		Apps    *cheapchartsApp `json:"apps"`
		MacApps *cheapchartsApp `json:"macapps"`
	} `json:"results"`
}

type cheapchartsApp struct {
	Title               string  `json:"title"`
	ItemStatus          string  `json:"itemStatus"`
	Price               float64 `json:"price"`
	PriceBefore         float64 `json:"priceBefore"`
	Currency            string  `json:"currency"`
	PriceDropIndicator  int     `json:"priceDropIndicator"`
	PriceLastChangeDate string  `json:"priceLastChangeDate"`
}

var appstorePriceRoute = routeutils.RouteSpec{
	Path:        "price/:country/:type/:id",
	Name:        "Price Drop Watcher",
	Example:     "appstore/price/us/ios/id1444383602",
	Maintainers: []string{"xihale"},
	Description: "Emits an item when a tracked App Store app's price drops (via CheapCharts)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("country", "App Store country code, e.g. us or cn"),
		routeutils.RequiredParam("type", "App type: ios or mac"),
		routeutils.RequiredParam("id", "App Store app id like id1444383602"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  AppstorePriceHandler,
}

func currencySymbol(code string) string {
	symbols := map[string]string{
		"USD": "$", "EUR": "€", "GBP": "£", "CNY": "¥", "JPY": "¥",
		"HKD": "HK$", "TWD": "NT$", "KRW": "₩", "AUD": "A$", "CAD": "CA$",
		"INR": "₹", "RUB": "₽", "BRL": "R$", "CHF": "CHF ", "SEK": "kr ",
	}
	if s, ok := symbols[strings.ToUpper(code)]; ok {
		return s
	}
	return code + " "
}

// AppstorePriceHandler handles /appstore/price/:country/:type/:id.
func AppstorePriceHandler(c *ctxpkg.Context) (*models.Feed, error) {
	country := strings.ToLower(c.Param("country"))
	appType := routeutils.ParseEnum(c.Param("type"), "apps", "ios", "mac")
	itemType := "apps"
	if appType == "mac" {
		itemType = "macapps"
	}
	id := strings.TrimPrefix(strings.TrimSpace(c.Param("id")), "id")

	link := fmt.Sprintf("https://apps.apple.com/%s/app/id%s", country, id)
	apiURL := fmt.Sprintf(
		"https://buster.cheapcharts.de/v1/DetailData.php?store=itunes&country=%s&itemType=%s&idInStore=%s",
		country, itemType, id)

	var resp cheapchartsResp
	if err := routeutils.GetJSONWithHeaders(c.Parent(), c.Client(), apiURL,
		map[string]string{"Referer": "http://www.cheapcharts.info/itunes/" + country + "/apps/detail-view/" + id},
		&resp); err != nil {
		return nil, err
	}
	result := resp.Results.Apps
	if itemType == "macapps" {
		result = resp.Results.MacApps
	}
	if result == nil || result.Title == "" {
		return nil, fmt.Errorf("app %s is not tracked by CheapCharts for country %s", id, country)
	}

	title := fmt.Sprintf("Price watcher: %s for %s", result.Title, map[string]string{"apps": "iOS", "macapps": "macOS"}[itemType])
	feed := routeutils.NewFeed(title, link, "Price drop alerts for "+result.Title)
	if result.PriceDropIndicator == -1 {
		desc := fmt.Sprintf(`<a href="%s">Go to App Store</a><br>Previous price: %s%.2f`, html.EscapeString(link),
			currencySymbol(result.Currency), result.PriceBefore)
		pubDate, _ := dateutil.ParseDate(result.PriceLastChangeDate)
		item := routeutils.NewItem(
			fmt.Sprintf("%s is now %s%.2f", result.Title, currencySymbol(result.Currency), result.Price),
			link, desc, pubDate)
		item.GUID = id + result.PriceLastChangeDate
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
