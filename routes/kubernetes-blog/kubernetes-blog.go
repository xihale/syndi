package routes

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var kubernetesBlogRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "Kubernetes Blog",
	Example:     "kubernetes-blog",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from the Kubernetes blog (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    2 * time.Hour,
	Handler:     KubernetesBlogHandler,
}

// KubernetesBlogHandler handles /kubernetes-blog
func KubernetesBlogHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://kubernetes.io/feed.xml")
	if err != nil {
		return nil, err
	}
	feed.Title = "Kubernetes Blog"
	feed.Link = "https://kubernetes.io/blog/"
	feed.Description = "Latest posts from the Kubernetes blog"
	return feed, nil
}
