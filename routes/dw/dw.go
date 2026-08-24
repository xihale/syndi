package routes

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

const dwFeedBase = "https://rss.dw.com/rdf"

// dwRDFItem models one top-level <item> of DW's RSS 1.0 (RDF) feed.
type dwRDFItem struct {
	Title          string `xml:"title"`
	Link           string `xml:"link"`
	Description    string `xml:"description"`
	DCDate         string `xml:"http://purl.org/dc/elements/1.1/ date"`
	DCSubject      string `xml:"http://purl.org/dc/elements/1.1/ subject"`
	DWSynContentID string `xml:"http://rss.dw.com/syndication/dwsyn/ contentID"`
}

// dwRDFDoc models DW's RSS 1.0 document; real <item> entries sit at the top level.
type dwRDFDoc struct {
	XMLName xml.Name    `xml:"RDF"`
	Title   string      `xml:"channel>title"`
	Link    string      `xml:"channel>link"`
	Desc    string      `xml:"channel>description"`
	Items   []dwRDFItem `xml:"item"`
}

var dwRSSRoute = routeutils.RouteSpec{
	Path:        "rss/:channel",
	Name:        "DW RSS Feed",
	Example:     "dw/rss/rss-en-all",
	Maintainers: []string{"xihale"},
	Description: "Deutsche Welle syndication feed; channel defaults to rss-en-all when set to '-'. Other languages work too, e.g. rss-chi-all (Chinese), rss-de-all (German)",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("channel", "Feed channel from https://corporate.dw.com/en/rss-feeds/a-68693346, e.g. rss-en-all, rss-chi-all, rss-de-all; use '-' for the English default"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  DWRSSHandler,
}

// DWRSSHandler handles /dw/rss/:channel
func DWRSSHandler(c *ctxpkg.Context) (*models.Feed, error) {
	channel := strings.TrimSpace(c.Param("channel"))
	if channel == "-" || channel == "" {
		channel = "rss-en-all"
	}
	feedURL := fmt.Sprintf("%s/%s", dwFeedBase, channel)

	data, err := routeutils.FetchBytes(c.Parent(), c.Client(), feedURL)
	if err != nil {
		return nil, err
	}
	var doc dwRDFDoc
	if err := routeutils.UnmarshalXML(data, &doc); err != nil {
		return nil, fmt.Errorf("parse DW RDF feed %s: %w", feedURL, err)
	}

	if doc.Title == "" {
		doc.Title = "Deutsche Welle"
	}
	feed := routeutils.NewFeed(doc.Title, stripDWTracking(doc.Link), doc.Desc)
	for i := range doc.Items {
		it := doc.Items[i]
		link := stripDWTracking(it.Link)
		title := strings.TrimSpace(it.Title)
		if title == "" && link == "" {
			continue
		}
		var pub time.Time
		if it.DCDate != "" {
			if t, err := dateutil.ParseDate(strings.TrimSpace(it.DCDate)); err == nil {
				pub = t
			}
		}
		item := routeutils.NewItem(title, link, strings.TrimSpace(it.Description), pub)
		if it.DWSynContentID != "" {
			item.GUID = "dw-" + it.DWSynContentID
		} else if link != "" {
			item.GUID = link
		}
		if s := strings.TrimSpace(it.DCSubject); s != "" {
			routeutils.SetCategories(item, s)
		}
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

// stripDWTracking removes the ?maca=... tracking parameter DW appends to links.
func stripDWTracking(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Del("maca")
	u.RawQuery = q.Encode()
	return u.String()
}
