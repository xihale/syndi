package routes

import (
	"time"

	"github.com/xihale/rsshub-go/internal/routeutils"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

var awsBlogRoute = routeutils.RouteSpec{
	Path:        "",
	Name:        "AWS News Blog",
	Example:     "aws-blog",
	Maintainers: []string{"xihale"},
	Description: "Latest posts from the AWS News Blog (native RSS, normalized)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	CacheTTL:    time.Hour,
	Handler:     AWSBlogHandler,
}

// AWSBlogHandler handles /aws-blog
func AWSBlogHandler(c *ctxpkg.Context) (*models.Feed, error) {
	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), "https://aws.amazon.com/blogs/aws/feed/")
	if err != nil {
		return nil, err
	}
	feed.Title = "AWS News Blog"
	feed.Link = "https://aws.amazon.com/blogs/aws/"
	feed.Description = "Latest posts from the AWS News Blog"
	return feed, nil
}
