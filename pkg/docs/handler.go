package docs

import (
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xihale/syndi/pkg/registry"
	"go.uber.org/zap"

	"github.com/xihale/syndi/pkg/logger"
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
	// /rss lists every available feed route (JSON catalog).
	engine.GET("/rss", h.RoutesJSONHandler)
	engine.NoRoute(h.DocsHandler)

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
	body := "# Syndi\n" +
		"User-agent: *\n" +
		"Allow: /\n" +
		"Disallow: /api/\n"
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, body)
}

func renderHTML(c *gin.Context, status int, tmpl *template.Template, data any) {
	// gin presets 404 for NoRoute handlers; make the intended status explicit.
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		// Headers may already be written; log and fall back to a plain error.
		logger.Error("template execute failed", zap.Error(err))
	}
}

// IndexHandler serves the main documentation page at "/".
// Legacy /?ns=x URLs redirect to /x.
func (h *Handler) IndexHandler(c *gin.Context) {
	if ns := c.Query("ns"); ns != "" {
		c.Redirect(http.StatusMovedPermanently, "/"+ns)
		return
	}
	h.renderIndex(c, "")
}

func (h *Handler) renderIndex(c *gin.Context, onlyNS string) {
	namespaces := h.docData.Namespaces
	if onlyNS != "" {
		var filtered []*NamespaceDoc
		for _, n := range namespaces {
			if strings.EqualFold(n.Name, onlyNS) {
				filtered = append(filtered, n)
			}
		}
		if len(filtered) == 0 {
			c.String(http.StatusNotFound, "404 namespace not found: %s", onlyNS)
			return
		}
		namespaces = filtered
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
		CrumbNS:     onlyNS,
	}

	renderHTML(c, http.StatusOK, h.indexTmpl, pageData)
}

// DocsHandler catches every non-API path and serves it as documentation:
//
//	/{ns}                       -> namespace overview
//	/{ns}/route/...             -> route doc (:id style params match literally)
//	/docs, /docs/route?path=    -> legacy redirects
func (h *Handler) DocsHandler(c *gin.Context) {
	p := c.Request.URL.Path

	switch p {
	case "/docs":
		c.Redirect(http.StatusMovedPermanently, "/")
		return
	case "/docs/route":
		if q := c.Query("path"); q != "" {
			c.Redirect(http.StatusMovedPermanently, q)
			return
		}
		c.Redirect(http.StatusMovedPermanently, "/")
		return
	}

	segs := splitSegs(p)
	if len(segs) == 0 {
		h.IndexHandler(c)
		return
	}

	ns := segs[0]
	if !h.hasNamespace(ns) {
		c.String(http.StatusNotFound, "404 page not found: %s", p)
		return
	}
	if len(segs) == 1 {
		// Bare-namespace paths may also BE a route ("/arstechnica").
		// An exact route match takes precedence over the namespace overview.
		if route := h.matchDocRoute("/" + ns); route != nil {
			h.renderRoutePage(c, route)
			return
		}
		h.renderIndex(c, ns)
		return
	}

	route := h.matchDocRoute("/" + strings.Join(segs, "/"))
	if route == nil {
		c.String(http.StatusNotFound, "404 route not found: %s", p)
		return
	}
	h.renderRoutePage(c, route)
}

func splitSegs(p string) []string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, s := range parts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (h *Handler) hasNamespace(name string) bool {
	for _, n := range h.docData.Namespaces {
		if strings.EqualFold(n.Name, name) {
			return true
		}
	}
	return false
}

// matchDocRoute finds the route whose pattern matches a concrete doc path,
// where :param matches any single segment and *wild matches the rest.
func (h *Handler) matchDocRoute(path string) *RouteDoc {
	for _, r := range h.docData.Routes {
		if matchPattern(r.Path, path) {
			return r
		}
	}
	return nil
}

func matchPattern(pattern, path string) bool {
	ps := strings.Split(strings.Trim(pattern, "/"), "/")
	xs := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range ps {
		if strings.HasPrefix(seg, "*") {
			return true
		}
		if i >= len(xs) {
			return false
		}
		if strings.HasPrefix(seg, ":") {
			continue
		}
		if seg != xs[i] {
			return false
		}
	}
	return len(ps) == len(xs)
}

// renderRoutePage renders the detail page for a route doc.
func (h *Handler) renderRoutePage(c *gin.Context, route *RouteDoc) {
	path := route.Path

	related := make([]*RouteDoc, 0)
	for _, r := range h.docData.Routes {
		if r.Path != path && strings.Split(r.Path, "/")[1] == strings.Split(path, "/")[1] {
			related = append(related, r)
		}
	}
	if len(related) > 5 {
		related = related[:5]
	}

	ns := strings.Split(path, "/")[1]
	routeEnvStatuses := make([]CredStatus, 0)
	for _, req := range registry.NamespaceEnvReqs(ns) {
		cs := CredStatus{Key: req.Key}
		value := os.Getenv(req.Key)
		for _, f := range req.Fields {
			cs.Fields = append(cs.Fields, CredField{
				Name:    f.Name,
				Present: strings.Contains(value, f.Name+"="),
			})
		}
		routeEnvStatuses = append(routeEnvStatuses, cs)
	}

	pageData := RoutePageData{
		Title:            route.Name,
		Route:            route,
		Related:          related,
		RouteEnvStatuses: routeEnvStatuses,
	}

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
