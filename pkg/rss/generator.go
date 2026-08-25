package rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/pkg/models"
)

// Namespace URIs used by the generators
const (
	NSAtom    = "http://www.w3.org/2005/Atom"
	NSContent = "http://purl.org/rss/1.0/modules/content/"
)

// Feed represents RSS 2.0 structure
type RSSFeed struct {
	XMLName      xml.Name    `xml:"rss"`
	Version      string      `xml:"version,attr"`
	XmlnsContent string      `xml:"xmlns:content,attr"`
	Channel      *RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title          string    `xml:"title"`
	Link           string    `xml:"link"`
	Description    string    `xml:"description"`
	Language       string    `xml:"language,omitempty"`
	LastBuildDate  string    `xml:"lastBuildDate,omitempty"`
	PubDate        string    `xml:"pubDate,omitempty"`
	WebMaster      string    `xml:"webMaster,omitempty"`
	ManagingEditor string    `xml:"managingEditor,omitempty"`
	Category       string    `xml:"category,omitempty"`
	Generator      string    `xml:"generator,omitempty"`
	Items          []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	Description string      `xml:"description,omitempty"`
	GUID        string      `xml:"guid"`
	PubDate     string      `xml:"pubDate,omitempty"`
	Author      string      `xml:"author,omitempty"`
	Categories  []string    `xml:"category,omitempty"`
	Content     *RSSContent `xml:"encoded,omitempty"`
}

// RSSContent is the item full-text element: the standard content module's
// <content:encoded> (recognized by Feedly/Inoreader/FreshRSS et al.), not a
// bare <content>.
type RSSContent struct {
	XMLName xml.Name `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",chardata"`
}

// AtomFeed represents Atom 1.0 structure
type AtomFeed struct {
	XMLName   xml.Name     `xml:"feed"`
	XMLNs     string       `xml:"xmlns,attr"`
	Title     string       `xml:"title"`
	ID        string       `xml:"id"`
	Updated   string       `xml:"updated"`
	Links     []AtomLink   `xml:"link"`
	Subtitle  string       `xml:"subtitle,omitempty"`
	Authors   []AtomAuthor `xml:"author,omitempty"`
	Generator string       `xml:"generator,omitempty"`
	Entries   []AtomEntry  `xml:"entry"`
}

type AtomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr,omitempty"`
}

type AtomAuthor struct {
	Name  string `xml:"name"`
	Email string `xml:"email,omitempty"`
	URI   string `xml:"uri,omitempty"`
}

type AtomEntry struct {
	Title      string       `xml:"title"`
	ID         string       `xml:"id"`
	Updated    string       `xml:"updated"`
	Published  string       `xml:"published,omitempty"`
	Link       []AtomLink   `xml:"link"`
	Summary    string       `xml:"summary,omitempty"`
	Content    AtomContent  `xml:"content,omitempty"`
	Authors    []AtomAuthor `xml:"author,omitempty"`
	Categories []string     `xml:"category,omitempty"`
}

type AtomContent struct {
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// GenerateRSS creates an RSS 2.0 feed
func GenerateRSS(feed *models.Feed) ([]byte, error) {
	updated := feedUpdated(feed)
	rss := &RSSFeed{
		Version:      "2.0",
		XmlnsContent: NSContent,
		Channel: &RSSChannel{
			Title:       feed.Title,
			Link:        feed.Link,
			Description: feed.Description,
			Generator:   "Syndi/0.0.1 (+https://github.com/xihale/syndi)",
		},
	}

	if feed.Author != nil && feed.Author.Email != "" {
		rss.Channel.ManagingEditor = feed.Author.Email
	}

	if !updated.IsZero() {
		rss.Channel.LastBuildDate = updated.UTC().Format(time.RFC1123Z)
	}

	if feed.Updated != nil && !feed.Updated.IsZero() {
		rss.Channel.PubDate = feed.Updated.Format(time.RFC1123Z)
	}

	for _, item := range feed.Items {
		rssItem := RSSItem{
			Title:       item.Title,
			Link:        item.Link,
			GUID:        item.GUID,
			Description: item.Description,
		}

		if !item.PubDate.IsZero() {
			rssItem.PubDate = item.PubDate.Format(time.RFC1123Z)
		}

		if item.Author != nil && item.Author.Email != "" {
			rssItem.Author = item.Author.Email
		}

		if len(item.Categories) > 0 {
			rssItem.Categories = item.Categories
		}

		if item.Description != "" {
			rssItem.Content = &RSSContent{
				Type:    "html",
				Content: item.Description,
			}
		}

		rss.Channel.Items = append(rss.Channel.Items, rssItem)
	}

	var buf strings.Builder
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(rss); err != nil {
		return nil, err
	}

	return withContentPrefix([]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!-- generator="Syndi/0.0.1" -->
%s`, buf.String()))), nil
}

// withContentPrefix rewrites the content module element from the
// default-namespace form encoding/xml can emit (<encoded xmlns="…">) to the
// conventional prefixed <content:encoded>. Item text is entity-escaped, so
// these literals cannot occur inside content.
func withContentPrefix(doc []byte) []byte {
	doc = bytes.ReplaceAll(doc,
		[]byte(`<encoded xmlns="`+NSContent+`"`),
		[]byte(`<content:encoded`))
	return bytes.ReplaceAll(doc, []byte(`</encoded>`), []byte(`</content:encoded>`))
}

// feedUpdated returns the effective last-modified time of a feed: its own
// Updated timestamp, else the newest item PubDate. It is derived from feed
// data (never time.Now()) so the same cached feed always serializes to
// identical bytes and ETags stay stable across requests.
func feedUpdated(feed *models.Feed) time.Time {
	if feed.Updated != nil && !feed.Updated.IsZero() {
		return *feed.Updated
	}
	var newest time.Time
	for _, item := range feed.Items {
		if item.PubDate.After(newest) {
			newest = item.PubDate
		}
	}
	return newest
}

// entryUpdated returns the effective updated time of an Atom entry.
func entryUpdated(item *models.Item, fallback time.Time) time.Time {
	if item.Updated != nil && !item.Updated.IsZero() {
		return item.Updated.UTC()
	}
	if !item.PubDate.IsZero() {
		return item.PubDate.UTC()
	}
	return fallback.UTC()
}

// GenerateAtom creates an Atom 1.0 feed
func GenerateAtom(feed *models.Feed) ([]byte, error) {
	updated := feedUpdated(feed)
	atom := &AtomFeed{
		XMLNs:     NSAtom,
		Title:     feed.Title,
		ID:        feed.Link,
		Generator: "Syndi/0.0.1 (+https://github.com/xihale/syndi)",
		Links: []AtomLink{
			{Rel: "self", Href: feed.Link, Type: "application/atom+xml"},
			{Rel: "alternate", Href: feed.Link, Type: "text/html"},
		},
	}

	if !updated.IsZero() {
		atom.Updated = updated.UTC().Format(time.RFC3339)
	}

	if feed.Author != nil {
		author := AtomAuthor{
			Name:  feed.Author.Name,
			Email: feed.Author.Email,
			URI:   feed.Author.URI,
		}
		if author.Name != "" || author.Email != "" || author.URI != "" {
			atom.Authors = append(atom.Authors, author)
		}
	}

	for _, item := range feed.Items {
		entry := AtomEntry{
			Title:   item.Title,
			ID:      item.GUID,
			Updated: entryUpdated(&item, updated).Format(time.RFC3339),
			Link:    []AtomLink{{Rel: "alternate", Href: item.Link}},
			Summary: item.Description,
		}

		if !item.PubDate.IsZero() {
			entry.Published = item.PubDate.UTC().Format(time.RFC3339)
		}

		if item.Author != nil {
			author := AtomAuthor{
				Name:  item.Author.Name,
				Email: item.Author.Email,
				URI:   item.Author.URI,
			}
			if author.Name != "" || author.Email != "" || author.URI != "" {
				entry.Authors = append(entry.Authors, author)
			}
		}

		if len(item.Categories) > 0 {
			entry.Categories = item.Categories
		}

		if item.Description != "" {
			entry.Content = AtomContent{
				Type:    "html",
				Content: item.Description,
			}
		}

		atom.Entries = append(atom.Entries, entry)
	}

	var buf strings.Builder
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(atom); err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!-- generator="Syndi/0.0.1" -->
%s`, buf.String())), nil
}
