// Package routes ports the RSSHub ithome (IT之家) namespace.
package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// ithomeRoot is the site origin. Tests swap it for a fixture server;
// production code must treat it as read-only.
var ithomeRoot = "https://www.ithome.com"

var ithomeProfile = disguise.Chrome().Lang("zh-CN,zh;q=0.9")

// ithomeCSTZone is Beijing time used by the article publish timestamps.
var ithomeCSTZone = time.FixedZone("UTC+8", 8*3600)

// ithomeListEntry is one list-level article reference shared by the ranking,
// tag and zt listings; detail pages enrich title/body/time afterwards.
type ithomeListEntry struct {
	Title   string
	Link    string
	Summary string
	PubDate time.Time
	Author  string
}

// ithomeGUIDPath normalizes an absolute ithome URL to its path part so GUIDs
// stay short but stable (e.g. "0/994/535.htm").
func ithomeGUIDPath(link string) string {
	path := strings.TrimPrefix(link, ithomeRoot)
	return strings.TrimPrefix(path, "/")
}

// parseITHomeLimit reads the optional limit query param shared by listings.
// An unset value yields def (0 = no limit); explicit values are capped at 50.
func parseITHomeLimit(c *ctxpkg.Context, def int) int {
	raw := strings.TrimSpace(c.QueryParam("limit"))
	if raw == "" {
		return def
	}
	return routeutils.ParsePositiveInt(raw, def, 50)
}

// ithomeFetchArticle fetches a news detail page and returns cleaned body HTML,
// the publish time and the author. Best-effort callers may ignore the error
// and keep the list-level summary only.
func ithomeFetchArticle(c *ctxpkg.Context, link string) (string, time.Time, string, error) {
	doc, err := ithomeProfile.Fetch(link).GetHTML(c.Parent(), c.Client())
	if err != nil {
		return "", time.Time{}, "", err
	}
	para := doc.FindSelector("#paragraph")
	if para == nil {
		return "", time.Time{}, "", fmt.Errorf("ithome: #paragraph not found on %s", link)
	}
	body, _ := para.Html()
	if cleaned, cerr := routeutils.CleanDescription(body, link, routeutils.DefaultCleanOptions()); cerr == nil && strings.TrimSpace(cleaned) != "" {
		body = cleaned
	}

	var pubDate time.Time
	if s := strings.TrimSpace(doc.Text("#pubtime_baidu")); s != "" {
		if t, perr := time.ParseInLocation("2006/1/2 15:04:05", s, ithomeCSTZone); perr == nil {
			pubDate = t
		}
	}
	author := strings.TrimSpace(doc.Text("#author_baidu strong"))
	if author == "" {
		author = strings.TrimSpace(doc.Text("#source_baidu strong"))
	}
	if body == "" {
		return "", pubDate, author, fmt.Errorf("ithome: empty #paragraph on %s", link)
	}
	return body, pubDate, author, nil
}

// ithomeAppendDetailed maps list entries into the feed, fetching each article
// for full content/time/author (list values serve as fallback on failures).
// When limit > 0 appending stops once the feed holds limit items.
func ithomeAppendDetailed(c *ctxpkg.Context, feed *models.Feed, guidPrefix string, entries []ithomeListEntry, limit int) {
	for _, e := range entries {
		if limit > 0 && len(feed.Items) >= limit {
			break
		}
		if e.Link == "" || e.Title == "" {
			continue
		}
		desc := ""
		if s := e.Summary; s != "" {
			desc = "<p>" + html.EscapeString(strings.TrimSpace(s)) + "</p>"
		}
		item := routeutils.NewItem(e.Title, e.Link, desc, e.PubDate)
		item.GUID = guidPrefix + ithomeGUIDPath(e.Link)
		if e.Author != "" {
			routeutils.SetItemAuthor(item, e.Author, "", "")
		}
		if body, pubDate, author, err := ithomeFetchArticle(c, e.Link); err == nil {
			item.Description = body
			if !pubDate.IsZero() {
				item.PubDate = pubDate
			}
			if author != "" {
				routeutils.SetItemAuthor(item, author, "", "")
			}
		}
		routeutils.AddItem(feed, item)
	}
}
