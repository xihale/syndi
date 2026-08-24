package routes

import (
	"fmt"
	"html"
	"sort"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const (
	usgsBaseURL  = "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/"
	usgsMaxItems = 100
	usgsMapURL   = "https://earthquake.usgs.gov/earthquakes/map/"
)

// usgsSummaryFeed mirrors the USGS earthquake summary GeoJSON feeds.
type usgsSummaryFeed struct {
	Features []usgsFeature `json:"features"`
	Metadata struct {
		Generated int64  `json:"generated"`
		Title     string `json:"title"`
		Count     int    `json:"count"`
	} `json:"metadata"`
}

type usgsFeature struct {
	ID         string         `json:"id"`
	Properties usgsProperties `json:"properties"`
	Geometry   struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
}

type usgsProperties struct {
	Mag     *float64 `json:"mag"`
	MagType string   `json:"magType"`
	Place   string   `json:"place"`
	Time    int64    `json:"time"` // epoch milliseconds
	URL     string   `json:"url"`
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Tsunami int      `json:"tsunami"`
}

// pubDate converts the epoch-millisecond origin time to time.Time.
func (f usgsFeature) pubDate() time.Time {
	if f.Properties.Time <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(f.Properties.Time)
}

// buildUSGSFeed fetches a USGS summary GeoJSON feed and maps it into a Feed,
// sorted by origin time descending and capped at 100 items.
func buildUSGSFeed(c *ctxpkg.Context, feedURL, title, description string) (*models.Feed, error) {
	var payload usgsSummaryFeed
	if err := routeutils.GetJSON(c.Parent(), c.Client(), feedURL, &payload); err != nil {
		return nil, err
	}

	features := payload.Features
	sort.SliceStable(features, func(i, j int) bool {
		return features[i].Properties.Time > features[j].Properties.Time
	})
	if len(features) > usgsMaxItems {
		features = features[:usgsMaxItems]
	}

	feed := routeutils.NewFeed(title, usgsMapURL, description)
	routeutils.AppendMappedItems(feed, features, 0, func(f usgsFeature) *models.Item {
		props := f.Properties
		if props.Title == "" && props.Place == "" {
			return nil
		}
		link := props.URL
		if link == "" {
			return nil
		}
		itemTitle := props.Title
		if itemTitle == "" {
			itemTitle = props.Place
		}

		mag := "unknown"
		if props.Mag != nil {
			mag = fmt.Sprintf("%.1f %s", *props.Mag, props.MagType)
		}

		desc := fmt.Sprintf("<b>Magnitude:</b> %s<br/><b>Place:</b> %s",
			html.EscapeString(mag), html.EscapeString(props.Place))
		if len(f.Geometry.Coordinates) >= 2 {
			desc += fmt.Sprintf("<br/><b>Coordinates:</b> %.4f lat, %.4f lon",
				f.Geometry.Coordinates[1], f.Geometry.Coordinates[0])
		}
		if props.Type != "" {
			desc += "<br/><b>Type:</b> " + html.EscapeString(props.Type)
		}
		if props.Tsunami == 1 {
			desc += "<br/><b>Tsunami flag:</b> yes"
		}

		item := routeutils.NewItem(itemTitle, link, desc, f.pubDate())
		if f.ID != "" {
			item.GUID = f.ID
		}
		return item
	})
	return feed, nil
}
