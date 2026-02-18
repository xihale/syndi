package routes

import (
	"fmt"
	"time"

	ctxpkg "github.com/rsshub/go/pkg/context"
	"github.com/rsshub/go/pkg/models"
	"github.com/rsshub/go/pkg/registry"
	"github.com/rsshub/go/internal/routeutils"
)

func init() {
	route := &models.Route{
		Path:         "/dribbble/:list",
		Name:         "Dribbble Popular Shots",
		Example:      "dribbble/popular",
		Maintainers:  []string{"yourname"},
		Description:  "Fetch popular shots from Dribbble",
		Categories:   []models.Category{{Name: "design"}},
		Features:     models.Features{},
		Handler:      DribbbleShotsHandler,
		Parameters: []models.Parameter{
			{Name: "list", Required: false, Description: "List type: popular, recent, following (default: popular)"},
		},
	}
	if err := registry.GetRegistry().Register(route); err != nil {
		panic(err)
	}
}

// DribbbleShotsHandler handles /dribbble/:list
func DribbbleShotsHandler(c *ctxpkg.Context) (*models.Feed, error) {
	list := c.Param("list")
	if list == "" {
		list = "popular"
	}

	ctx := c.Parent()

	url := fmt.Sprintf("https://api.dribbble.com/v2/user/shots?list=%s&per_page=30", list)

	var shots []DribbbleShot
	if err := routeutils.GetJSON(ctx, c.Client(), url, &shots); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(
		fmt.Sprintf("Dribbble %s Shots", list),
		fmt.Sprintf("https://dribbble.com/shots?list=%s", list),
		fmt.Sprintf("Popular shots from Dribbble (%s list)", list),
	)

	for _, shot := range shots {
		description := fmt.Sprintf(
			"<img src=\"%s\"/><br/>%s",
			shot.Images.Normal,
			shot.Description,
		)

		item := routeutils.NewItem(
			shot.Title,
			shot.HTMLURL,
			description,
			shot.CreatedAt,
		)
		item.GUID = fmt.Sprintf("dribbble-%d", shot.ID)

		// Set author
		if shot.User.Name != "" {
			routeutils.SetAuthor(item, shot.User.Name,
				routeutils.WithAuthorURI(shot.User.HTMLURL))
		}

		// Add categories (tags)
		if len(shot.Tags) > 0 {
			categories := make([]string, len(shot.Tags))
			for i, tag := range shot.Tags {
				categories[i] = tag
			}
			routeutils.SetCategories(item, categories...)
		}

		routeutils.AddItem(feed, item)
	}

	return feed, nil
}

type DribbbleShot struct {
	ID          int               `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	HTMLURL     string            `json:"html_url"`
	Images      DribbbleImages    `json:"images"`
	CreatedAt   time.Time         `json:"created_at"`
	User        DribbbleUser      `json:"user"`
	Tags        []string          `json:"tags"`
}

type DribbbleImages struct {
	Hidpi string `json:"hidpi"`
	Normal string `json:"normal"`
	Teaser string `json:"teaser"`
}

type DribbbleUser struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Username string `json:"username"`
	HTMLURL string `json:"html_url"`
}
