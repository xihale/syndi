package routes

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var homebrewFormulaRoute = routeutils.RouteSpec{
	Path:        "formula/:formula",
	Name:        "Homebrew Formula Updates",
	Example:     "homebrew/formula/wget",
	Maintainers: []string{"xihale"},
	Description: "Fetch version bump history for a Homebrew formula",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("formula", "Formula name, e.g. wget"),
	},
	CacheTTL: 2 * time.Hour,
	Handler:  HomebrewFormulaHandler,
}

// HomebrewFormulaHandler handles /homebrew/formula/:formula
func HomebrewFormulaHandler(c *ctxpkg.Context) (*models.Feed, error) {
	formula := c.Param("formula")
	if formula == "" {
		return nil, fmt.Errorf("formula parameter is required")
	}
	ctx := c.Parent()

	// Formulas live at Formula/<first-letter>/<formula>.rb in homebrew-core.
	path := fmt.Sprintf("Formula/%s/%s.rb", strings.ToLower(formula[:1]), formula)
	url := fmt.Sprintf(
		"https://api.github.com/repos/Homebrew/homebrew-core/commits?path=%s&per_page=20",
		path,
	)

	var commits []homebrewCommit
	if err := routeutils.GetJSON(ctx, c.Client(), url, &commits); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Homebrew formula %s", formula),
		fmt.Sprintf("https://formulae.brew.sh/formula/%s", formula),
		fmt.Sprintf("Recent commits touching the %s formula", formula),
	)
	routeutils.AppendMappedItems(feed, commits, 20, func(commit homebrewCommit) *models.Item {
		if commit.SHA == "" {
			return nil
		}
		link := fmt.Sprintf("https://github.com/Homebrew/homebrew-core/commit/%s", commit.SHA)
		title := firstLineOf(commit.Commit.Message)
		if title == "" {
			title = commit.SHA[:7]
		}

		item := routeutils.NewItem(title, link, html.EscapeString(title), commit.Commit.Committer.Date)
		item.GUID = commit.SHA
		if commit.Author != nil && commit.Author.Login != "" {
			routeutils.SetItemAuthor(item, commit.Author.Login, "", fmt.Sprintf("https://github.com/%s", commit.Author.Login))
		}
		routeutils.SetCategories(item, formula)
		return item
	})

	return feed, nil
}

type homebrewCommit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Message string `json:"message"`
		Author  struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
}

func firstLineOf(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
