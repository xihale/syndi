package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	dateutil "github.com/xihale/rsshub-go/pkg/utils/date"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

const bioRxivDaysBack = 7

var biorxivLatestRoute = routeutils.RouteSpec{
	Path:        "latest",
	Name:        "bioRxiv Latest Papers",
	Example:     "biorxiv/latest",
	Maintainers: []string{"xihale"},
	Description: "Latest preprints posted to bioRxiv in the last 7 days",
	Categories:  []models.Category{{Name: "study"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    1 * time.Hour,
	Handler:     BiorxivLatestHandler,
}

// BiorxivLatestHandler handles /biorxiv/latest
func BiorxivLatestHandler(c *ctxpkg.Context) (*models.Feed, error) {
	return biorxivMedrxivLatest(c, "biorxiv", "bioRxiv", "https://www.biorxiv.org/")
}

type biorxivResp struct {
	Messages []struct {
		Status string `json:"status"`
	} `json:"messages"`
	Collection []biorxivPost `json:"collection"`
}

type biorxivPost struct {
	Title    string `json:"title"`
	Authors  string `json:"authors"`
	DOI      string `json:"doi"`
	Date     string `json:"date"`
	Abstract string `json:"abstract"`
	Category string `json:"category"`
	JXRURL   string `json:"jxr_url"`
}

func (p *biorxivPost) link() string {
	if p.JXRURL != "" {
		return p.JXRURL
	}
	if p.DOI != "" {
		return "https://doi.org/" + p.DOI
	}
	return ""
}

// biorxivMedrxivLatest powers both /biorxiv/latest and /medrxiv/latest.
// The API only accepts explicit YYYY-MM-DD/YYYY-MM-DD intervals (a bare day
// count returns "Both dates must be in yyyy-mm-dd format").
func biorxivMedrxivLatest(c *ctxpkg.Context, server, label, siteLink string) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 30, 100)
	ctx := c.Parent()

	end := time.Now().UTC()
	start := end.AddDate(0, 0, -bioRxivDaysBack)
	rangeParam := fmt.Sprintf("%s/%s", start.Format("2006-01-02"), end.Format("2006-01-02"))

	var posts []biorxivPost
	for cursor := 0; len(posts) < limit && cursor < 4; cursor++ {
		url := fmt.Sprintf("https://api.biorxiv.org/details/%s/%s/%d", server, rangeParam, cursor)
		var resp biorxivResp
		if err := routeutils.GetJSON(ctx, c.Client(), url, &resp); err != nil {
			return nil, err
		}
		if len(resp.Collection) == 0 {
			break
		}
		posts = append(posts, resp.Collection...)
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("%s Latest Papers", label),
		siteLink,
		fmt.Sprintf("Preprints posted to %s in the last %d days", label, bioRxivDaysBack),
	)

	for _, p := range posts {
		link := p.link()
		title := strings.TrimSpace(p.Title)
		if title == "" || link == "" {
			continue
		}
		pubDate, _ := dateutil.ParseDate(p.Date)

		var b strings.Builder
		b.WriteString("<p>" + html.EscapeString(strings.TrimSpace(p.Abstract)) + "</p>")
		if authors := strings.TrimSpace(p.Authors); authors != "" {
			b.WriteString("<br/><em>Authors:</em> " + html.EscapeString(authors))
		}

		item := routeutils.NewItem(title, link, b.String(), pubDate)
		item.GUID = p.DOI
		routeutils.SetCategories(item, p.Category)
		routeutils.AddItem(feed, item)
	}

	return feed, nil
}
