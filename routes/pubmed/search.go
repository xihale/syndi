package routes

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

// eutilsUA is the Tooling User-Agent NCBI asks automated clients to send.
const eutilsUA = "rsshub-go/1.0 (https://github.com/xihale/syndi; Tooling)"

const eutilsBaseURL = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils"

var pubmedSearchRoute = routeutils.RouteSpec{
	Path:        "search/:term",
	Name:        "PubMed Search",
	Example:     "pubmed/search/crispr",
	Maintainers: []string{"xihale"},
	Description: "Latest PubMed results for a search term (newest first)",
	Categories:  []models.Category{{Name: "study"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("term", "search term, e.g. crispr"),
		routeutils.OptionalParam("limit", "number of results, default 20, max 100"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  PubmedSearchHandler,
}

// PubmedSearchHandler handles /pubmed/search/:term
func PubmedSearchHandler(c *ctxpkg.Context) (*models.Feed, error) {
	term := strings.TrimSpace(c.Param("term"))
	if term == "" {
		return nil, fmt.Errorf("search term is required")
	}
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 100)
	ctx := c.Parent()
	headers := map[string]string{"User-Agent": eutilsUA}

	// Step 1: esearch to get the newest PMIDs.
	esearchURL := fmt.Sprintf("%s/esearch.fcgi?db=pubmed&term=%s&retmax=%d&retmode=json&sort=date", eutilsBaseURL, url.QueryEscape(term), limit)
	var searchResp pubmedSearchResp
	if err := routeutils.GetJSONWithHeaders(ctx, c.Client(), esearchURL, headers, &searchResp); err != nil {
		return nil, err
	}
	ids := searchResp.ESearchResult.IDList
	if len(ids) == 0 {
		return routeutils.NewFeed(
			fmt.Sprintf("PubMed: %s", term),
			fmt.Sprintf("https://pubmed.ncbi.nlm.nih.gov/?term=%s", url.QueryEscape(term)),
			fmt.Sprintf("Latest PubMed results for %q", term),
		), nil
	}

	// Step 2: esummary for document metadata.
	esummaryURL := fmt.Sprintf("%s/esummary.fcgi?db=pubmed&id=%s&retmode=json", eutilsBaseURL, strings.Join(ids, ","))
	var raw struct {
		Result map[string]json.RawMessage `json:"result"`
	}
	if err := routeutils.GetJSONWithHeaders(ctx, c.Client(), esummaryURL, headers, &raw); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("PubMed: %s", term),
		fmt.Sprintf("https://pubmed.ncbi.nlm.nih.gov/?term=%s", url.QueryEscape(term)),
		fmt.Sprintf("Latest PubMed results for %q", term),
	)

	var uidList []string
	if blob, ok := raw.Result["uids"]; ok {
		if err := json.Unmarshal(blob, &uidList); err != nil {
			return nil, fmt.Errorf("failed to parse esummary uid list: %w", err)
		}
	}

	for _, uid := range uidList {
		var doc pubmedDoc
		blob, ok := raw.Result[uid]
		if !ok {
			continue
		}
		if err := json.Unmarshal(blob, &doc); err != nil {
			continue
		}
		title := strings.TrimSpace(doc.Title)
		if title == "" || uid == "" {
			continue
		}
		title = strings.TrimSuffix(title, ".")

		link := fmt.Sprintf("https://pubmed.ncbi.nlm.nih.gov/%s/", uid)
		pubDate, _ := dateutil.ParseDate(doc.SortPubDate)
		if pubDate.IsZero() {
			pubDate, _ = dateutil.ParseDate(doc.PubDate)
		}

		item := routeutils.NewItem(title, link, buildPubmedDescription(doc), pubDate)
		item.GUID = uid
		if len(doc.Authors) > 0 && doc.Authors[0].Name != "" {
			routeutils.SetItemAuthor(item, doc.Authors[0].Name, "", "")
			names := make([]string, 0, len(doc.Authors))
			for _, a := range doc.Authors {
				if a.Name != "" {
					names = append(names, a.Name)
				}
			}
			if len(names) > 1 {
				item.Description += "<br/><em>Authors:</em> " + html.EscapeString(strings.Join(names, ", "))
			}
		}
		if journal := strings.TrimSpace(doc.FullJournalName); journal != "" {
			routeutils.SetCategories(item, journal)
		}
		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

func buildPubmedDescription(d pubmedDoc) string {
	var b strings.Builder
	b.WriteString("<p><em>Journal:</em> " + html.EscapeString(strings.TrimSpace(d.FullJournalName)) + "</p>")
	if pubDate := strings.TrimSpace(d.PubDate); pubDate != "" {
		b.WriteString("<p><em>Published:</em> " + html.EscapeString(pubDate) + "</p>")
	}
	return b.String()
}

type pubmedSearchResp struct {
	ESearchResult struct {
		IDList []string `json:"idlist"`
	} `json:"esearchresult"`
}

type pubmedDoc struct {
	Title           string `json:"title"`
	PubDate         string `json:"pubdate"`
	SortPubDate     string `json:"sortpubdate"`
	FullJournalName string `json:"fulljournalname"`
	ELocationID     string `json:"elocationid"`
	Authors         []struct {
		Name string `json:"name"`
	} `json:"authors"`
}
