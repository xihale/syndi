package routes

import (
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// youtubeProfile is used only for the handle -> channel id resolution scrape;
// the native video feeds themselves are fetched without any disguise.
var youtubeProfile = disguise.Chrome()

// resolveYouTubeHandle maps a @handle to its UC... channel id by scraping the
// public channel page HTML (the native feed API only accepts channel ids).
func resolveYouTubeHandle(c *ctxpkg.Context, handle string) (string, error) {
	handle = strings.TrimPrefix(strings.TrimSpace(handle), "@")
	page := "https://www.youtube.com/@" + handle
	pageHTML, err := youtubeProfile.Fetch(page).GetString(c.Parent(), c.Client())
	if err != nil {
		return "", fmt.Errorf("failed to load %s: %w", page, err)
	}
	m := ytChannelIDRe.FindStringSubmatch(pageHTML)
	if len(m) < 2 {
		return "", fmt.Errorf("could not resolve YouTube handle @%s to a channel id", handle)
	}
	return m[1], nil
}

var ytChannelIDRe = regexp.MustCompile(`"externalId":"(UC[a-zA-Z0-9_-]{5,})"`)

// Native YouTube Atom feed served at https://www.youtube.com/feeds/videos.xml.
type ytAtomFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Title   string       `xml:"title"`
	Links   []ytAtomLink `xml:"link"`
	Author  struct {
		Name string `xml:"name"`
		URI  string `xml:"uri"`
	} `xml:"author"`
	Entries []ytAtomEntry `xml:"entry"`
}

type ytAtomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type ytAtomEntry struct {
	ID        string       `xml:"id"`
	VideoID   string       `xml:"videoId"`
	ChannelID string       `xml:"channelId"`
	Title     string       `xml:"title"`
	Links     []ytAtomLink `xml:"link"`
	Published time.Time    `xml:"published"`
	Updated   time.Time    `xml:"updated"`
	Group     struct {
		Description string `xml:"description"`
		Thumbnail   struct {
			URL string `xml:"url,attr"`
		} `xml:"thumbnail"`
		Community struct {
			StarRating struct {
				Count int64 `xml:"count,attr"`
			} `xml:"starRating"`
			Statistics struct {
				Views int64 `xml:"views,attr"`
			} `xml:"statistics"`
		} `xml:"community"`
	} `xml:"group"`
}

func (e *ytAtomEntry) altLink() string {
	for _, l := range e.Links {
		if l.Rel == "alternate" || l.Rel == "" {
			return l.Href
		}
	}
	if len(e.Links) > 0 {
		return e.Links[0].Href
	}
	return ""
}

func youTubeAlternate(feed *ytAtomFeed) string {
	for _, l := range feed.Links {
		if l.Rel == "alternate" {
			return l.Href
		}
	}
	return ""
}

// fetchYouTubeVideos loads the native Atom feed for a channel or playlist
// query ("channel_id=..." / "playlist_id=...") and normalizes it.
func fetchYouTubeVideos(c *ctxpkg.Context, query string) (*models.Feed, error) {
	feedURL := "https://www.youtube.com/feeds/videos.xml?" + query
	var atom ytAtomFeed
	if err := routeutils.GetXML(c.Parent(), c.Client(), feedURL, &atom); err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", feedURL, err)
	}
	title := atom.Title
	if title == "" {
		title = atom.Author.Name
	}
	feed := routeutils.NewFeed(title, youTubeAlternate(&atom), "Latest videos from "+title)
	for _, e := range atom.Entries {
		itemLink := e.altLink()
		if e.Title == "" || itemLink == "" {
			continue
		}
		desc := fmt.Sprintf(
			`<iframe src="https://www.youtube.com/embed/%s" style="width:100%%;max-width:720px;aspect-ratio:16/9;" frameborder="0" allowfullscreen></iframe>`,
			e.VideoID,
		)
		if e.Group.Description != "" {
			desc += "<br>" + html.EscapeString(e.Group.Description)
		}
		if e.Group.Community.Statistics.Views > 0 {
			desc += fmt.Sprintf("<br>Views: %d", e.Group.Community.Statistics.Views)
			if e.Group.Community.StarRating.Count > 0 {
				desc += fmt.Sprintf(" | Ratings: %d", e.Group.Community.StarRating.Count)
			}
		}
		item := routeutils.NewItem(e.Title, itemLink, desc, e.Published)
		item.GUID = e.ID
		routeutils.SetItemAuthor(item, atom.Author.Name, "", atom.Author.URI)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
