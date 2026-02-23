package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
)

type finding struct {
	level   string
	route   string
	message string
}

const (
	minCacheTTL = 30 * time.Second
	maxCacheTTL = 30 * 24 * time.Hour
)

func main() {
	strict := flag.Bool("strict", false, "treat warnings as errors")
	flag.Parse()

	registerRoutePackages()

	routes := registry.GetRegistry().GetAllRoutes()
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Path < routes[j].Path
	})

	findings := make([]finding, 0)
	if len(routes) == 0 {
		findings = append(findings, finding{
			level:   "error",
			route:   "<registry>",
			message: "no routes registered",
		})
	}

	for _, route := range routes {
		findings = append(findings, validateRoute(route)...)
	}

	errorCount := 0
	warnCount := 0
	for _, item := range findings {
		switch item.level {
		case "error":
			errorCount++
			fmt.Printf("ERROR [%s] %s\n", item.route, item.message)
		case "warn":
			warnCount++
			fmt.Printf("WARN  [%s] %s\n", item.route, item.message)
		}
	}

	fmt.Printf("\nVerified %d routes. errors=%d warnings=%d\n", len(routes), errorCount, warnCount)

	if errorCount > 0 || (*strict && warnCount > 0) {
		os.Exit(1)
	}
}

func validateRoute(route *models.Route) []finding {
	if route == nil {
		return []finding{{
			level:   "error",
			route:   "<nil>",
			message: "route is nil",
		}}
	}

	findings := make([]finding, 0)
	routeID := route.Path
	if strings.TrimSpace(routeID) == "" {
		routeID = "<empty-path>"
	}

	if strings.TrimSpace(route.Path) == "" {
		findings = append(findings, finding{"error", routeID, "path is required"})
	} else if !strings.HasPrefix(route.Path, "/") {
		findings = append(findings, finding{"error", routeID, "path must start with '/'"})
	}

	if strings.TrimSpace(route.Name) == "" {
		findings = append(findings, finding{"error", routeID, "name is required"})
	}
	if route.Handler == nil {
		findings = append(findings, finding{"error", routeID, "handler is required"})
	}

	namespace := routeNamespace(route.Path)
	if strings.TrimSpace(route.Description) == "" {
		findings = append(findings, finding{"warn", routeID, "description is empty"})
	}
	example := strings.TrimSpace(route.Example)
	if example == "" {
		findings = append(findings, finding{"warn", routeID, "example is empty"})
	} else {
		if strings.HasPrefix(example, "/") {
			findings = append(findings, finding{"warn", routeID, "example should not start with '/'"})
		}
		if strings.Contains(example, "://") {
			findings = append(findings, finding{"warn", routeID, "example should not be a URL"})
		}
		if strings.Contains(example, ":") {
			findings = append(findings, finding{"warn", routeID, "example should not include path parameter placeholders"})
		}
		exampleTrimmed := strings.TrimPrefix(example, "/")
		if namespace != "" && exampleTrimmed != namespace && !strings.HasPrefix(exampleTrimmed, namespace+"/") {
			findings = append(findings, finding{"warn", routeID, fmt.Sprintf("example should start with namespace '%s'", namespace)})
		}
	}
	if len(route.Categories) == 0 {
		findings = append(findings, finding{"warn", routeID, "categories are empty"})
	} else {
		seenCategories := make(map[string]struct{}, len(route.Categories))
		for _, category := range route.Categories {
			name := strings.TrimSpace(category.Name)
			if name == "" {
				findings = append(findings, finding{"warn", routeID, "category with empty name"})
				continue
			}
			key := strings.ToLower(name)
			if _, exists := seenCategories[key]; exists {
				findings = append(findings, finding{"warn", routeID, fmt.Sprintf("duplicate category '%s'", name)})
				continue
			}
			seenCategories[key] = struct{}{}
		}
	}

	if namespace != "test" {
		if len(route.Maintainers) == 0 {
			findings = append(findings, finding{"warn", routeID, "maintainers are empty"})
		}
		for _, maintainer := range route.Maintainers {
			if strings.EqualFold(strings.TrimSpace(maintainer), "yourname") {
				findings = append(findings, finding{"warn", routeID, "maintainer contains placeholder 'yourname'"})
			}
		}
	}

	if namespace != "test" && route.CacheTTL != nil {
		if *route.CacheTTL < minCacheTTL {
			findings = append(findings, finding{"warn", routeID, fmt.Sprintf("cache TTL %s is very short", route.CacheTTL.String())})
		}
		if *route.CacheTTL > maxCacheTTL {
			findings = append(findings, finding{"warn", routeID, fmt.Sprintf("cache TTL %s is unusually long", route.CacheTTL.String())})
		}
	}

	paramIndex := make(map[string]models.Parameter, len(route.Parameters))
	for _, param := range route.Parameters {
		name := strings.TrimSpace(param.Name)
		if name == "" {
			findings = append(findings, finding{"error", routeID, "parameter with empty name"})
			continue
		}
		if _, exists := paramIndex[name]; exists {
			findings = append(findings, finding{"error", routeID, fmt.Sprintf("duplicate parameter '%s'", name)})
			continue
		}
		if strings.TrimSpace(param.Description) == "" {
			findings = append(findings, finding{"warn", routeID, fmt.Sprintf("parameter '%s' has empty description", name)})
		}
		paramIndex[name] = param
	}

	for _, dup := range duplicatePathParams(route.Path) {
		findings = append(findings, finding{"warn", routeID, fmt.Sprintf("path parameter '%s' appears multiple times", dup)})
	}

	for _, pathParam := range extractPathParams(route.Path) {
		param, exists := paramIndex[pathParam]
		if !exists {
			findings = append(findings, finding{"error", routeID, fmt.Sprintf("missing parameter metadata for path param '%s'", pathParam)})
			continue
		}
		if !param.Required {
			findings = append(findings, finding{"warn", routeID, fmt.Sprintf("path param '%s' should usually be marked Required=true", pathParam)})
		}
	}

	return findings
}

func extractPathParams(path string) []string {
	parts := strings.Split(path, "/")
	seen := make(map[string]struct{})
	params := make([]string, 0)
	for _, part := range parts {
		if len(part) < 2 || !strings.HasPrefix(part, ":") {
			continue
		}
		name := strings.TrimPrefix(part, ":")
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		params = append(params, name)
	}
	return params
}

func duplicatePathParams(path string) []string {
	parts := strings.Split(path, "/")
	seen := make(map[string]struct{})
	dupes := make([]string, 0)
	for _, part := range parts {
		if len(part) < 2 || !strings.HasPrefix(part, ":") {
			continue
		}
		name := strings.TrimPrefix(part, ":")
		if _, exists := seen[name]; exists {
			dupes = append(dupes, name)
			continue
		}
		seen[name] = struct{}{}
	}
	return dupes
}

func routeNamespace(path string) string {
	path = strings.TrimPrefix(strings.TrimSpace(path), "/")
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "/"); idx > 0 {
		return path[:idx]
	}
	return path
}
