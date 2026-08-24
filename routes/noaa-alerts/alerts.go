package routes

import (
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const (
	noaaAlertsURL = "https://api.weather.gov/alerts/active"
	noaaMaxAlerts = 20
	noaaContactUA = "rsshub-go/1.0 (+https://github.com/xihale/syndi; contact via https://github.com/xihale/syndi/issues)"
)

var noaaAlertsRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "NOAA Active Weather Alerts",
	Example:     "noaa-alerts",
	Maintainers: []string{"xihale"},
	Description: "Active US National Weather Service alerts (NOAA/NWS)",
	Categories:  []models.Category{{Name: "science"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    15 * time.Minute,
	Handler:     NOAAAlertsHandler,
}

// NOAAAlertsHandler handles /noaa-alerts.
// api.weather.gov requests clients identify themselves with a contact User-Agent.
func NOAAAlertsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	var resp noaaAlertFeed
	if err := disguise.Custom(noaaContactUA).Fetch(noaaAlertsURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		"NWS Active Alerts",
		"https://alerts.weather.gov/",
		"Active weather alerts from the US National Weather Service (NOAA)",
	)

	routeutils.AppendMappedItems(feed, resp.Features, noaaMaxAlerts, func(f noaaAlertFeature) *models.Item {
		props := f.Properties
		title := props.Headline
		if title == "" {
			title = props.Event
		}
		if title == "" {
			return nil
		}
		link := props.URL // canonical https://api.weather.gov/alerts/... URL from "@id"
		if link == "" {
			return nil
		}

		var b strings.Builder
		b.WriteString("<b>" + html.EscapeString(title) + "</b>")
		if props.AreaDesc != "" {
			b.WriteString("<br/><b>Areas:</b> " + html.EscapeString(props.AreaDesc))
		}
		meta := make([]string, 0, 4)
		if props.Severity != "" {
			meta = append(meta, "Severity: "+props.Severity)
		}
		if props.Certainty != "" {
			meta = append(meta, "Certainty: "+props.Certainty)
		}
		if props.Urgency != "" {
			meta = append(meta, "Urgency: "+props.Urgency)
		}
		if !props.Expires.IsZero() {
			meta = append(meta, "Expires: "+props.Expires.UTC().Format(time.RFC3339))
		}
		if len(meta) > 0 {
			b.WriteString("<br/>" + html.EscapeString(strings.Join(meta, " | ")))
		}
		if props.Description != "" {
			b.WriteString("<br/><br/>" + strings.ReplaceAll(html.EscapeString(props.Description), "\n", "<br/>"))
		}
		if props.Instruction != "" {
			b.WriteString("<br/><br/><b>Instructions:</b><br/>" + strings.ReplaceAll(html.EscapeString(props.Instruction), "\n", "<br/>"))
		}

		item := routeutils.NewItem(title, link, b.String(), props.Sent)
		if props.AlertID != "" {
			item.GUID = props.AlertID
		}
		if props.SenderName != "" {
			routeutils.SetItemAuthor(item, props.SenderName, "", "")
		}
		return item
	})
	return feed, nil
}

// noaaAlertFeed mirrors the NWS alert GeoJSON feed. Note: the upstream API does
// not accept a limit query parameter, so we cap client-side.
type noaaAlertFeed struct {
	Features []noaaAlertFeature `json:"features"`
}

type noaaAlertFeature struct {
	ID         string         `json:"id"`
	Properties noaaAlertProps `json:"properties"`
}

type noaaAlertProps struct {
	AlertID     string    `json:"id"`  // urn:oid:... alert identifier
	URL         string    `json:"@id"` // canonical API URL for this alert
	Event       string    `json:"event"`
	Headline    string    `json:"headline"`
	AreaDesc    string    `json:"areaDesc"`
	Severity    string    `json:"severity"`
	Certainty   string    `json:"certainty"`
	Urgency     string    `json:"urgency"`
	SenderName  string    `json:"senderName"`
	Sent        time.Time `json:"sent"`    // RFC3339 with offset, decodes directly
	Expires     time.Time `json:"expires"` // RFC3339 with offset, decodes directly
	Description string    `json:"description"`
	Instruction string    `json:"instruction"`
}
