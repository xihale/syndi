package docs

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
)

// DocData represents the complete documentation structure
type DocData struct {
	Namespaces []*NamespaceDoc `json:"namespaces"`
	Routes     []*RouteDoc     `json:"routes"`
	Categories []string        `json:"categories"`
	Total      int             `json:"total"`
}

// NamespaceDoc represents a namespace documentation
type NamespaceDoc struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	RouteCount  int         `json:"route_count"`
	Routes      []*RouteDoc `json:"routes"`
}

// RouteDoc represents a route documentation
type RouteDoc struct {
	Path        string          `json:"path"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Example     string          `json:"example"`
	ExampleURL  string          `json:"example_url"`
	Categories  []string        `json:"categories"`
	Parameters  []*ParameterDoc `json:"parameters"`
	Features    *FeaturesDoc    `json:"features,omitempty"`
	Maintainers []string        `json:"maintainers"`
	CacheTTL    string          `json:"cache_ttl"`
	CurlExample string          `json:"curl_example"`

	// EnvDeps lists credential env vars this route requires.
	EnvDeps []string `json:"env_deps,omitempty"`
	// MissingDeps/Unavailable are resolved once at startup against the live
	// process environment; Unavailable routes render grayed out.
	MissingDeps []string `json:"-"`
	Unavailable bool     `json:"-"`
}

// ParameterDoc represents a route parameter documentation
type ParameterDoc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
}

// FeaturesDoc represents route features
type FeaturesDoc struct {
	SupportRadar bool `json:"support_radar"`
	AntiCrawler  bool `json:"anti_crawler"`
}

// Generate generates complete documentation from registry
func Generate() *DocData {
	reg := registry.GetRegistry()

	doc := &DocData{
		Namespaces: make([]*NamespaceDoc, 0),
		Routes:     make([]*RouteDoc, 0),
		Categories: make([]string, 0),
	}

	// Track categories
	categoryMap := make(map[string]bool)

	// Process namespaces
	namespaces := reg.GetAllNamespaces()
	sort.Slice(namespaces, func(i, j int) bool {
		return namespaces[i].Name < namespaces[j].Name
	})

	for _, ns := range namespaces {
		nsDoc := &NamespaceDoc{
			Name:        ns.Name,
			Description: fmt.Sprintf("Routes related to %s", ns.Name),
			RouteCount:  len(ns.Routes),
			Routes:      make([]*RouteDoc, 0),
		}

		for _, route := range ns.Routes {
			routeDoc := routeToDoc(route)
			nsDoc.Routes = append(nsDoc.Routes, routeDoc)
			doc.Routes = append(doc.Routes, routeDoc)

			// Track categories
			for _, cat := range route.Categories {
				categoryMap[cat.Name] = true
			}
		}

		// Sort routes within namespace
		sort.Slice(nsDoc.Routes, func(i, j int) bool {
			return nsDoc.Routes[i].Path < nsDoc.Routes[j].Path
		})

		doc.Namespaces = append(doc.Namespaces, nsDoc)
	}

	// Extract categories
	for cat := range categoryMap {
		doc.Categories = append(doc.Categories, cat)
	}
	sort.Strings(doc.Categories)

	doc.Total = len(doc.Routes)

	return doc
}

// routeToDoc converts a Route to RouteDoc
func routeToDoc(route *models.Route) *RouteDoc {
	// Build example URL
	exampleURL := route.Path
	if route.Example != "" {
		exampleURL = "/" + route.Example
	}

	// Build parameters
	params := make([]*ParameterDoc, 0)
	for _, p := range route.Parameters {
		params = append(params, &ParameterDoc{
			Name:        p.Name,
			Description: p.Description,
			Required:    p.Required,
			Type:        "string",
		})
	}

	// Build features
	var features *FeaturesDoc
	if route.Features.SupportRadar || route.Features.AntiCrawler {
		features = &FeaturesDoc{
			SupportRadar: route.Features.SupportRadar,
			AntiCrawler:  route.Features.AntiCrawler,
		}
	}

	// Build cache TTL string
	cacheTTL := "default (15m)"
	if route.CacheTTL != nil {
		cacheTTL = formatTTL(*route.CacheTTL)
	}

	// Build categories
	categories := make([]string, 0)
	for _, cat := range route.Categories {
		categories = append(categories, cat.Name)
	}

	// Build curl example
	curlExample := buildCurlExample(exampleURL)

	return &RouteDoc{
		Path:        route.Path,
		Name:        route.Name,
		Description: route.Description,
		Example:     route.Example,
		ExampleURL:  exampleURL,
		Categories:  categories,
		Parameters:  params,
		Features:    features,
		Maintainers: route.Maintainers,
		CacheTTL:    cacheTTL,
		CurlExample: curlExample,
		EnvDeps:     route.Features.EnvDeps,
	}
}

// formatTTL renders a duration without zero components: 1h0m0s -> 1h.
func formatTTL(d time.Duration) string {
	var b strings.Builder
	if h := int(d.Hours()); h > 0 {
		b.WriteString(strconv.Itoa(h) + "h")
	}
	if m := int(d.Minutes()) % 60; m > 0 {
		b.WriteString(strconv.Itoa(m) + "m")
	}
	if sec := int(d.Seconds()) % 60; sec > 0 {
		b.WriteString(strconv.Itoa(sec) + "s")
	}
	if b.Len() == 0 {
		return d.String()
	}
	return b.String()
}

// buildCurlExample builds a curl command example
func buildCurlExample(path string) string {
	// Convert path to URL format
	urlPath := path
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	return fmt.Sprintf("curl http://localhost:1200%s", urlPath)
}

// GetByPath returns route documentation by path
func GetByPath(path string) *RouteDoc {
	reg := registry.GetRegistry()
	route := reg.GetRoute(path)
	if route == nil {
		return nil
	}
	return routeToDoc(route)
}

// GetByNamespace returns namespace documentation
func GetByNamespace(namespace string) *NamespaceDoc {
	reg := registry.GetRegistry()
	routes := reg.GetNamespaceRoutes(namespace)
	if routes == nil {
		return nil
	}

	nsDoc := &NamespaceDoc{
		Name:        namespace,
		Description: fmt.Sprintf("Routes related to %s", namespace),
		RouteCount:  len(routes),
		Routes:      make([]*RouteDoc, 0),
	}

	for _, route := range routes {
		nsDoc.Routes = append(nsDoc.Routes, routeToDoc(route))
	}

	sort.Slice(nsDoc.Routes, func(i, j int) bool {
		return nsDoc.Routes[i].Path < nsDoc.Routes[j].Path
	})

	return nsDoc
}

// Search routes by keyword
func Search(query string) []*RouteDoc {
	doc := Generate()
	results := make([]*RouteDoc, 0)
	query = strings.ToLower(query)

	for _, route := range doc.Routes {
		if strings.Contains(strings.ToLower(route.Name), query) ||
			strings.Contains(strings.ToLower(route.Description), query) ||
			strings.Contains(strings.ToLower(route.Path), query) {
			results = append(results, route)
		}
	}

	return results
}

// GetByCategory returns routes by category
func GetByCategory(category string) []*RouteDoc {
	doc := Generate()
	results := make([]*RouteDoc, 0)

	for _, route := range doc.Routes {
		for _, cat := range route.Categories {
			if strings.EqualFold(cat, category) {
				results = append(results, route)
				break
			}
		}
	}

	return results
}
