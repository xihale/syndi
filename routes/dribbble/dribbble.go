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

// dribbbleProfile: the shots listing pages sit behind a JS challenge for
// non-browser clients; a realistic Chrome header set gets the user page
// through. (Popular/search listings are challenge-gated even with browser
// headers and are therefore not ported.)
var dribbbleProfile = disguise.Chrome()

// dribbbleUserShots scrapes the public profile page of a Dribbble user or team.
func dribbbleUserShots(c *ctxpkg.Context, name string) (*models.Feed, error) {
	pageURL := fmt.Sprintf("https://dribbble.com/%s", name)
	doc, err := dribbbleProfile.Fetch(pageURL).GetHTML(c.Parent(), c.Client())
	if err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed("Dribbble - "+name, pageURL, "Latest shots of "+name)
	doc.Find("div.shot-thumbnail-base, li.shot-thumbnail").Each(func(_ int, s *goquery.Selection) {
		link, _ := s.Find(`a[href^="/shots/"]`).First().Attr("href")
		if link == "" {
			return
		}
		if !strings.HasPrefix(link, "http") {
			link = "https://dribbble.com" + link
		}

		title := strings.TrimSpace(s.Find(".shot-title").First().Text())
		if title == "" {
			title = strings.TrimSpace(s.Find("img").First().AttrOr("alt", ""))
		}
		if title == "" {
			title = link[strings.LastIndex(link, "/")+1:]
		}

		img := s.Find("img").First().AttrOr("data-src", "")
		if img == "" {
			img = s.Find("noscript img").First().AttrOr("src", "")
		}
		video := firstNonEmpty(
			s.AttrOr("data-video-teaser-large", ""),
			s.AttrOr("data-video-teaser-medium", ""),
			s.AttrOr("data-video-teaser-small", ""),
		)

		var desc string
		switch {
		case video != "":
			desc = fmt.Sprintf(
				`<video src="%s" poster="%s" controls loop muted style="max-width:100%%"></video>`,
				html.EscapeString(video), html.EscapeString(img),
			)
		case img != "":
			desc = fmt.Sprintf(`<img src="%s" alt="">`, html.EscapeString(img))
		default:
			desc = html.EscapeString(title)
		}

		routeutils.AddItem(feed, routeutils.NewItem(title, link, desc, time.Time{}))
	})
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no shots found for Dribbble user %q (page may be blocked)", name)
	}
	return feed, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
