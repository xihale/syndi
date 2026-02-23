package main

import (
	"strings"
	"testing"
	"time"

	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/models"
)

func dummyHandler(*ctxpkg.Context) (*models.Feed, error) {
	return nil, nil
}

func TestExtractPathParams(t *testing.T) {
	got := extractPathParams("/github/repos/:owner/:repo/:owner")
	if len(got) != 2 {
		t.Fatalf("expected 2 params, got %d (%v)", len(got), got)
	}
	if got[0] != "owner" || got[1] != "repo" {
		t.Fatalf("unexpected params order/content: %v", got)
	}
}

func TestRouteNamespace(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/github/repos/:owner", want: "github"},
		{path: "reddit/:subreddit", want: "reddit"},
		{path: "/", want: ""},
	}

	for _, tc := range tests {
		if got := routeNamespace(tc.path); got != tc.want {
			t.Fatalf("routeNamespace(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestValidateRouteValid(t *testing.T) {
	route := &models.Route{
		Path:        "/github/repos/:owner/:repo",
		Name:        "GitHub Repository Releases",
		Example:     "github/repos/DIYgod/RSSHub",
		Maintainers: []string{"xihale"},
		Description: "Fetch releases from a GitHub repository",
		Categories:  []models.Category{{Name: "dev"}},
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Repository owner"},
			{Name: "repo", Required: true, Description: "Repository name"},
		},
		Handler: dummyHandler,
	}

	findings := validateRoute(route)
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %v", findings)
	}
}

func TestValidateRouteMissingPathParamMetadata(t *testing.T) {
	route := &models.Route{
		Path:        "/github/repos/:owner/:repo",
		Name:        "Route",
		Example:     "x",
		Maintainers: []string{"xihale"},
		Description: "desc",
		Categories:  []models.Category{{Name: "dev"}},
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Owner"},
		},
		Handler: dummyHandler,
	}

	findings := validateRoute(route)
	if !hasFinding(findings, "error", "missing parameter metadata for path param 'repo'") {
		t.Fatalf("expected missing path param metadata error, got %v", findings)
	}
}

func TestValidateRoutePlaceholderMaintainer(t *testing.T) {
	route := &models.Route{
		Path:        "/github/repos/:owner",
		Name:        "Route",
		Example:     "x",
		Maintainers: []string{"yourname"},
		Description: "desc",
		Categories:  []models.Category{{Name: "dev"}},
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Owner"},
		},
		Handler: dummyHandler,
	}

	findings := validateRoute(route)
	if !hasFinding(findings, "warn", "placeholder 'yourname'") {
		t.Fatalf("expected placeholder maintainer warning, got %v", findings)
	}
}

func TestValidateRouteTestNamespaceSkipsMaintainerCheck(t *testing.T) {
	route := &models.Route{
		Path:        "/test/cache",
		Name:        "Cache Test",
		Example:     "test/cache",
		Maintainers: []string{"yourname"},
		Description: "desc",
		Categories:  []models.Category{{Name: "Test"}},
		Handler:     dummyHandler,
	}

	findings := validateRoute(route)
	if hasFinding(findings, "warn", "placeholder 'yourname'") {
		t.Fatalf("did not expect placeholder warning for /test namespace, got %v", findings)
	}
}

func TestValidateRouteExampleWarnings(t *testing.T) {
	route := &models.Route{
		Path:        "/github/repos/:owner",
		Name:        "Route",
		Example:     "/github/repos/:owner",
		Maintainers: []string{"xihale"},
		Description: "desc",
		Categories:  []models.Category{{Name: "dev"}},
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Owner"},
		},
		Handler: dummyHandler,
	}

	findings := validateRoute(route)
	if !hasFinding(findings, "warn", "example should not start with") {
		t.Fatalf("expected example leading slash warning, got %v", findings)
	}
	if !hasFinding(findings, "warn", "example should not include path parameter placeholders") {
		t.Fatalf("expected example placeholder warning, got %v", findings)
	}
}

func TestValidateRouteExampleNamespaceWarning(t *testing.T) {
	route := &models.Route{
		Path:        "/github/repos/:owner",
		Name:        "Route",
		Example:     "npm/react",
		Maintainers: []string{"xihale"},
		Description: "desc",
		Categories:  []models.Category{{Name: "dev"}},
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Owner"},
		},
		Handler: dummyHandler,
	}

	findings := validateRoute(route)
	if !hasFinding(findings, "warn", "example should start with namespace") {
		t.Fatalf("expected example namespace warning, got %v", findings)
	}
}

func TestValidateRouteDuplicateCategoryWarning(t *testing.T) {
	route := &models.Route{
		Path:        "/github/repos/:owner",
		Name:        "Route",
		Example:     "github/repos/octocat",
		Maintainers: []string{"xihale"},
		Description: "desc",
		Categories:  []models.Category{{Name: "Dev"}, {Name: "dev"}},
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Owner"},
		},
		Handler: dummyHandler,
	}

	findings := validateRoute(route)
	if !hasFinding(findings, "warn", "duplicate category") {
		t.Fatalf("expected duplicate category warning, got %v", findings)
	}
}

func TestValidateRouteParameterDescriptionWarning(t *testing.T) {
	route := &models.Route{
		Path:        "/github/repos/:owner",
		Name:        "Route",
		Example:     "github/repos/octocat",
		Maintainers: []string{"xihale"},
		Description: "desc",
		Categories:  []models.Category{{Name: "dev"}},
		Parameters: []models.Parameter{
			{Name: "owner", Required: true},
		},
		Handler: dummyHandler,
	}

	findings := validateRoute(route)
	if !hasFinding(findings, "warn", "empty description") {
		t.Fatalf("expected parameter description warning, got %v", findings)
	}
}

func TestValidateRouteDuplicatePathParamWarning(t *testing.T) {
	route := &models.Route{
		Path:        "/github/:owner/:owner",
		Name:        "Route",
		Example:     "github/octocat/again",
		Maintainers: []string{"xihale"},
		Description: "desc",
		Categories:  []models.Category{{Name: "dev"}},
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Owner"},
		},
		Handler: dummyHandler,
	}

	findings := validateRoute(route)
	if !hasFinding(findings, "warn", "appears multiple times") {
		t.Fatalf("expected duplicate path param warning, got %v", findings)
	}
}

func TestValidateRouteCacheTTLWarning(t *testing.T) {
	shortTTL := 10 * time.Second
	route := &models.Route{
		Path:        "/github/repos/:owner",
		Name:        "Route",
		Example:     "github/repos/octocat",
		Maintainers: []string{"xihale"},
		Description: "desc",
		Categories:  []models.Category{{Name: "dev"}},
		Parameters: []models.Parameter{
			{Name: "owner", Required: true, Description: "Owner"},
		},
		Handler:  dummyHandler,
		CacheTTL: &shortTTL,
	}

	findings := validateRoute(route)
	if !hasFinding(findings, "warn", "cache TTL") {
		t.Fatalf("expected cache TTL warning, got %v", findings)
	}
}

func hasFinding(findings []finding, level, messageContains string) bool {
	for _, item := range findings {
		if item.level == level && strings.Contains(item.message, messageContains) {
			return true
		}
	}
	return false
}
