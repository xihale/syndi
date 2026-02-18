package docs

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

	return &Handler{
		indexTmpl: indexTmpl,
		routeTmpl: routeTmpl,
		docData:   docData,
	}, nil
}

// RegisterRoutes registers documentation routes with Gin
func (h *Handler) RegisterRoutes(engine *gin.Engine) {
	// HTML documentation
	engine.GET("/docs", h.IndexHandler)
	engine.GET("/docs/route", h.RouteHandler)

	// JSON API
	engine.GET("/api/routes", h.RoutesJSONHandler)
	engine.GET("/api/routes/*path", h.RouteJSONHandler)
	engine.GET("/api/namespaces", h.NamespacesJSONHandler)
	engine.GET("/api/categories", h.CategoriesJSONHandler)
}

// IndexHandler serves the main documentation page
func (h *Handler) IndexHandler(c *gin.Context) {
	pageData := PageData{
		Title:       "All Routes",
		Namespaces:  h.docData.Namespaces,
		TotalRoutes: h.docData.Total,
		Categories:  h.docData.Categories,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.indexTmpl.Execute(c.Writer, pageData); err != nil {
		c.HTML(http.StatusInternalServerError, "error", gin.H{"error": err.Error()})
	}
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
		c.HTML(http.StatusNotFound, "error", gin.H{"error": "Route not found"})
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

	// Limit to 5 related routes
	if len(related) > 5 {
		related = related[:5]
	}

	pageData := RoutePageData{
		Route:   route,
		Related: related,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.routeTmpl.Execute(c.Writer, pageData); err != nil {
		c.HTML(http.StatusInternalServerError, "error", gin.H{"error": err.Error()})
	}
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
		"total": len(routes),
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
		"total": len(h.docData.Namespaces),
		"namespaces": h.docData.Namespaces,
	})
}

// CategoriesJSONHandler returns all categories as JSON
func (h *Handler) CategoriesJSONHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"total": len(h.docData.Categories),
		"categories": h.docData.Categories,
	})
}
