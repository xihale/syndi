package routeutils

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/pkg/models"
	"github.com/xihale/syndi/pkg/registry"
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

// RegisterRoute validates and registers a route with an explicit path.
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

// RegisterRouteWithBase resolves the spec path against a base and registers it.
// If spec.Path is absolute (starts with '/'), the base is ignored.
func RegisterRouteWithBase(spec RouteSpec, basePath string) error {
	resolved, err := resolveRoutePath(basePath, spec.Path)
	if err != nil {
		return err
	}
	spec.Path = resolved
	return RegisterRoute(spec)
}

// RegisterRoutesWithBase resolves and registers all specs against a base path.
func RegisterRoutesWithBase(basePath string, specs []RouteSpec) error {
	for i, spec := range specs {
		if err := RegisterRouteWithBase(spec, basePath); err != nil {
			return fmt.Errorf("route[%d]: %w", i, err)
		}
	}
	return nil
}

// MustRegisterRoute registers a route and panics on error.
func MustRegisterRoute(spec RouteSpec) {
	if err := RegisterRoute(spec); err != nil {
		panic(err)
	}
}

// MustRegisterRoutesWithBase registers routes and panics on error.
func MustRegisterRoutesWithBase(basePath string, specs []RouteSpec) {
	if err := RegisterRoutesWithBase(basePath, specs); err != nil {
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

// RequireParams flips the named parameters of spec to Required=true. Use it
// on deeper-path clones whose added :segment matches a parameter that the
// base route declared as optional (gin always requires the segment).
func RequireParams(spec RouteSpec, names ...string) RouteSpec {
	for i := range spec.Parameters {
		for _, n := range names {
			if spec.Parameters[i].Name == n {
				spec.Parameters[i].Required = true
			}
		}
	}
	return spec
}

func resolveRoutePath(basePath, path string) (string, error) {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "/") {
		return path, nil
	}

	base := strings.Trim(strings.TrimSpace(basePath), "/")
	if base == "" {
		if path == "" {
			return "", fmt.Errorf("route path is required")
		}
		return "/" + strings.Trim(path, "/"), nil
	}

	path = strings.Trim(path, "/")
	if path == "" {
		return "/" + base, nil
	}
	return "/" + base + "/" + path, nil
}
