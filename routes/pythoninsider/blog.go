package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var pythonInsiderRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Python Insider",
	Example:     "pythoninsider",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from Python Insider, the core developers' blog (native Atom feed, normalized)",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     PythonInsiderHandler,
}

// PythonInsiderHandler handles /pythoninsider
func PythonInsiderHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://pythoninsider.blogspot.com/feeds/posts/default")
	if err != nil {
		return nil, err
	}
	feed.Title = "Python Insider"
	feed.Link = "https://pythoninsider.blogspot.com/"
	feed.Description = "Python Insider - Python core development news and releases"
	return feed, nil
}
