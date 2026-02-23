package routeutils

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
)

// RouteSpec defines route metadata in a compact form.
type RouteSpec struct {
	Path        string
	Name        string
	URL         string
	Example     string
	Maintainers []string
	Description string
	Categories  []models.Category
	Features    models.Features
	Parameters  []models.Parameter
	CacheTTL    time.Duration
	Handler     models.HandlerFunc
}

// RegisterRoute validates and registers a route.
func RegisterRoute(spec RouteSpec) error {
	if strings.TrimSpace(spec.Path) == "" {
		return fmt.Errorf("route path is required")
	}
	if spec.Handler == nil {
		return fmt.Errorf("route handler is required for %s", spec.Path)
	}

	route := &models.Route{
		Path:        spec.Path,
		Name:        spec.Name,
		URL:         spec.URL,
		Example:     spec.Example,
		Maintainers: spec.Maintainers,
		Description: spec.Description,
		Categories:  spec.Categories,
		Features:    spec.Features,
		Parameters:  spec.Parameters,
		Handler:     spec.Handler,
	}
	if spec.CacheTTL > 0 {
		ttl := spec.CacheTTL
		route.CacheTTL = &ttl
	}

	return registry.GetRegistry().Register(route)
}

// MustRegisterRoute registers a route and panics on error.
func MustRegisterRoute(spec RouteSpec) {
	if err := RegisterRoute(spec); err != nil {
		panic(err)
	}
}

// RequiredParam creates a required route parameter.
func RequiredParam(name, description string) models.Parameter {
	return models.Parameter{
		Name:        name,
		Required:    true,
		Description: description,
	}
}

// OptionalParam creates an optional route parameter.
func OptionalParam(name, description string) models.Parameter {
	return models.Parameter{
		Name:        name,
		Required:    false,
		Description: description,
	}
}
