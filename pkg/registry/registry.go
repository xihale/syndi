package registry

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xihale/syndi/pkg/models"
)

// Registry manages route registration and discovery
type Registry struct {
	mu         sync.RWMutex
	routes     map[string]*models.Route
	catches    map[string][]*models.Route
	namespaces map[string]*Namespace
}

// Namespace represents a group of routes
type Namespace struct {
	Name        string
	Description string
	Routes      []*models.Route
}

// Global registry instance
var globalRegistry *Registry
var once sync.Once

// GetRegistry returns the global registry instance
func GetRegistry() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			routes:     make(map[string]*models.Route),
			catches:    make(map[string][]*models.Route),
			namespaces: make(map[string]*Namespace),
		}
	})
	return globalRegistry
}

// Register adds a route to the registry
func (r *Registry) Register(route *models.Route) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Use path as key
	key := route.Path
	if _, exists := r.routes[key]; exists {
		return fmt.Errorf("route already registered: %s", key)
	}

	// Extract namespace from path (first path segment)
	namespace := route.Path
	// Skip leading slash
	if strings.HasPrefix(namespace, "/") {
		namespace = namespace[1:]
	}
	// Get first path segment (before second slash)
	if slash := strings.Index(namespace, "/"); slash > 0 {
		namespace = namespace[:slash]
	}

	r.routes[key] = route

	// Add to namespace
	if ns, exists := r.namespaces[namespace]; exists {
		ns.Routes = append(ns.Routes, route)
	} else {
		r.namespaces[namespace] = &Namespace{
			Name:   namespace,
			Routes: []*models.Route{route},
		}
	}

	// Add to catch map
	r.catches[namespace] = append(r.catches[namespace], route)

	fmt.Printf("Registered route: %s - %s\n", route.Path, route.Name)
	return nil
}

// GetRoute retrieves a route by path
func (r *Registry) GetRoute(path string) *models.Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.routes[path]
}

// GetAllRoutes returns all registered routes
func (r *Registry) GetAllRoutes() []*models.Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]*models.Route, 0, len(r.routes))
	for _, route := range r.routes {
		routes = append(routes, route)
	}
	return routes
}

// GetNamespaceRoutes returns routes in a namespace
func (r *Registry) GetNamespaceRoutes(namespace string) []*models.Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if ns, ok := r.namespaces[namespace]; ok {
		return ns.Routes
	}
	return nil
}

// GetAllNamespaces returns all namespaces
func (r *Registry) GetAllNamespaces() []*Namespace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	namespaces := make([]*Namespace, 0, len(r.namespaces))
	for _, ns := range r.namespaces {
		namespaces = append(namespaces, ns)
	}
	return namespaces
}

// FindRoutes finds routes matching a path pattern
func (r *Registry) FindRoutes(path string) []*models.Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*models.Route
	for _, route := range r.routes {
		if matchPath(route.Path, path) {
			matches = append(matches, route)
		}
	}
	return matches
}

func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}
	// Simple pattern matching - could be enhanced with regex
	return strings.HasPrefix(path, pattern)
}
