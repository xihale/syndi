package routeutils

import (
	"context"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/parser/rssfeed"
	"github.com/xihale/syndi/pkg/models"
)

// GetFeed fetches a native RSS/Atom/RDF feed URL and parses it into models.Feed.
// This is the one-liner helper for "native feed wrapper" routes.
func GetFeed(ctx context.Context, cl *client.Client, url string) (*models.Feed, error) {
	data, err := cl.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	return rssfeed.Parse(data)
}

// GetFeedWithHeaders fetches a native feed with custom headers and parses it.
func GetFeedWithHeaders(ctx context.Context, cl *client.Client, url string, headers map[string]string) (*models.Feed, error) {
	data, err := cl.GetWithHeaders(ctx, url, headers)
	if err != nil {
		return nil, err
	}
	return rssfeed.Parse(data)
}
