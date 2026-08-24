package docs

import (
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xihale/rsshub-go/pkg/registry"
	"go.uber.org/zap"

	"github.com/xihale/rsshub-go/pkg/logger"
)

// Handler handles documentation HTTP requests
type Handler struct {
	indexTmpl *template.Template
	routeTmpl *template.Template
	docData   *DocData
}

// NewHandler creates a new documentation handler
func NewHandler() (*Handler, error) {
	indexTmpl, routeTmpl := ParseTemplates()
	docData := Generate()

	// Resolve per-route availability against the live process environment.
	// Env vars cannot change during the process lifetime in practice, so
	// computing once here is enough.
	configured := map[string]bool{}
	for _, statuses := range registry.AllEnvStatuses() {
		for _, st := range statuses {
			configured[st.Key] = st.Configured
		}
	}
	for _, r := range docData.Routes {
		for _, k := range r.EnvDeps {
			if !configured[k] {
				r.Unavailable = true
				r.MissingDeps = append(r.MissingDeps, k)
			}
		}
	}

	return &Handler{
		indexTmpl: indexTmpl,
		routeTmpl: routeTmpl,
		docData:   docData,
	}, nil
}

// RegisterRoutes registers documentation routes with Gin
func (h *Handler) RegisterRoutes(engine *gin.Engine) {
	// HTML documentation; root serves the index like upstream RSSHub.
	engine.GET("/", h.IndexHandler)
	engine.GET("/docs", h.IndexHandler)
	engine.GET("/docs/route", h.RouteHandler)

	// Plain-text endpoints
	engine.GET("/robots.txt", h.RobotsHandler)

	// JSON API
	engine.GET("/api/routes", h.RoutesJSONHandler)
	engine.GET("/api/routes/*path", h.RouteJSONHandler)
	engine.GET("/api/namespaces", h.NamespacesJSONHandler)
	engine.GET("/api/categories", h.CategoriesJSONHandler)
	engine.GET("/api/config", h.ConfigJSONHandler)
}

// RobotsHandler serves /robots.txt. Feeds are the public product, so crawlers
// may fetch everything except internal JSON APIs.
func (h *Handler) RobotsHandler(c *gin.Context) {
	body := "# RSSHub Go\n" +
		"User-agent: *\n" +
		"Allow: /\n" +
		"Disallow: /api/\n" +
		"Disallow: /docs/route\n"
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, body)
}

func renderHTML(c *gin.Context, status int, tmpl *template.Template, data any) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		// Headers may already be written; log and fall back to a plain error.
		logger.Error("template execute failed", zap.Error(err))
	}
}

// IndexHandler serves the main documentation page (also mounted at "/")
func (h *Handler) IndexHandler(c *gin.Context) {
	namespaces := h.docData.Namespaces
	if ns := c.Query("ns"); ns != "" {
		var filtered []*NamespaceDoc
		for _, n := range namespaces {
			if strings.EqualFold(n.Name, ns) {
				filtered = append(filtered, n)
			}
		}
		if len(filtered) > 0 {
			namespaces = filtered
		}
	}

	title := "All Routes"
	if len(namespaces) == 1 {
		title = namespaces[0].Name
	}

	pageData := PageData{
		Title:       title,
		Namespaces:  namespaces,
		TotalRoutes: h.docData.Total,
		Categories:  h.docData.Categories,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	renderHTML(c, http.StatusOK, h.indexTmpl, pageData)
}

// RouteHandler serves a single route documentation page
func (h *Handler) RouteHandler(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.Redirect(http.StatusFound, "/docs")
		return
	}

	route := GetByPath(path)
	if route == nil {
		c.String(http.StatusNotFound, "404 route not found: %s", path)
		return
	}

	// Find related routes (same namespace or category)
	related := make([]*RouteDoc, 0)
	for _, r := range h.docData.Routes {
		if r.Path != path {
			// Same namespace (first path segment)
			if strings.Split(r.Path, "/")[1] == strings.Split(path, "/")[1] {
				related = append(related, r)
			}
		}
	}

	// Credential requirements declared by this route's namespace.
	ns := strings.Split(path, "/")[1]
	routeEnvStatuses := make([]CredStatus, 0)
	for _, req := range registry.NamespaceEnvReqs(ns) {
		cs := CredStatus{Key: req.Key}
		value := os.Getenv(req.Key)
		for _, f := range req.Fields {
			// Cookie strings look like "z_c0=xxx; other=yyy".
			cs.Fields = append(cs.Fields, CredField{
				Name:    f.Name,
				Present: strings.Contains(value, f.Name+"="),
			})
		}
		routeEnvStatuses = append(routeEnvStatuses, cs)
	}

	// Limit to 5 related routes
	if len(related) > 5 {
		related = related[:5]
	}

	pageData := RoutePageData{
		Title:            route.Name,
		Route:            route,
		Related:          related,
		RouteEnvStatuses: routeEnvStatuses,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	renderHTML(c, http.StatusOK, h.routeTmpl, pageData)
}

// RoutesJSONHandler returns all routes as JSON
func (h *Handler) RoutesJSONHandler(c *gin.Context) {
	// Support filtering
	query := c.Query("q")
	category := c.Query("category")

	var routes []*RouteDoc
	if query != "" {
		routes = Search(query)
	} else if category != "" {
		routes = GetByCategory(category)
	} else {
		routes = h.docData.Routes
	}

	c.JSON(http.StatusOK, gin.H{
		"total":  len(routes),
		"routes": routes,
	})
}

// RouteJSONHandler returns a single route as JSON
func (h *Handler) RouteJSONHandler(c *gin.Context) {
	path := c.Param("path")
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	if path == "" || path == "routes" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter required"})
		return
	}

	route := GetByPath("/" + path)
	if route == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	c.JSON(http.StatusOK, route)
}

// NamespacesJSONHandler returns all namespaces as JSON
func (h *Handler) NamespacesJSONHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"total":      len(h.docData.Namespaces),
		"namespaces": h.docData.Namespaces,
	})
}

// CategoriesJSONHandler returns all categories as JSON
func (h *Handler) CategoriesJSONHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"total":      len(h.docData.Categories),
		"categories": h.docData.Categories,
	})
}

// ConfigJSONHandler reports which credential environment variables are set.
// Only boolean state is exposed; values are never echoed back.
func (h *Handler) ConfigJSONHandler(c *gin.Context) {
	all := registry.AllEnvStatuses()
	groups := make([]gin.H, 0, len(all))
	total := 0
	configured := 0
	for ns, statuses := range all {
		total += len(statuses)
		items := make([]gin.H, 0, len(statuses))
		for _, st := range statuses {
			if st.Configured {
				configured++
			}
			items = append(items, gin.H{
				"key":         st.Key,
				"description": st.Description,
				"scope":       st.Scope,
				"configured":  st.Configured,
				"fields":      st.Fields,
			})
		}
		groups = append(groups, gin.H{"namespace": ns, "env": items})
	}
	c.JSON(http.StatusOK, gin.H{
		"total":      total,
		"configured": configured,
		"namespaces": groups,
	})
}
