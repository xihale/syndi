package routes

import (
	"context"
	"os"
	"strings"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/routeutils"
)

// gitHubAPIBase is the REST API base shared by all GitHub routes. It is a
// variable so tests can point it at a fixture server.
var gitHubAPIBase = "https://api.github.com"

// gitHubWebBase is the main website base used to build fallback links.
const gitHubWebBase = "https://github.com"

// gitHubAPIOrigin is the fixed public REST origin. Unlike gitHubAPIBase it
// never changes and is used to normalize response URLs back into ids.
const gitHubAPIOrigin = "https://api.github.com"

// gitHubAPIHeaders returns the headers for GitHub API requests. Anonymous
// access is limited to 60 requests/hour; setting the optional GITHUB_TOKEN
// environment variable raises this to the authenticated quota.
func gitHubAPIHeaders() map[string]string {
	headers := map[string]string{
		"Accept": "application/vnd.github+json",
	}
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	return headers
}

// gitHubGetJSON fetches a GitHub API URL with standard headers.
func gitHubGetJSON(ctx context.Context, client *client.Client, url string, target interface{}) error {
	return routeutils.GetJSONWithHeaders(ctx, client, url, gitHubAPIHeaders(), target)
}
