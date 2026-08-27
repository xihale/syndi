package docs

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xihale/syndi/pkg/logger"
)

// assets is the embedded template/CSS/JS tree served by the docs UI.
//
//go:embed assets
var assets embed.FS

// docTemplates holds both page entrypoints; they share the layout shell and
// all partials, with each page file contributing its own "content" on top of
// a clone of the base set.
type docTemplates struct {
	index *template.Template
	route *template.Template
}

// parseDocTemplates loads templates/layout/base.html plus every partial into
// a base set, then clones it once per page under assets/templates/pages.
func parseDocTemplates() (*docTemplates, error) {
	base, err := template.New("docs").Funcs(templateFuncs()).ParseFS(assets,
		"assets/templates/layout/base.html",
		"assets/templates/partials/*.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse base templates: %w", err)
	}

	index := template.Must(base.Clone())
	if _, err := index.ParseFS(assets, "assets/templates/pages/index.html"); err != nil {
		return nil, fmt.Errorf("parse index page: %w", err)
	}
	route := template.Must(base.Clone())
	if _, err := route.ParseFS(assets, "assets/templates/pages/route.html"); err != nil {
		return nil, fmt.Errorf("parse route page: %w", err)
	}

	return &docTemplates{
		index: index.Lookup("shell"),
		route: route.Lookup("shell"),
	}, nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"lower":    strings.ToLower,
		"seg1":     seg1,
		"pathHTML": pathHTML,
	}
}

// seg1 returns the first path segment (the namespace) of a route path.
func seg1(p string) string {
	parts := strings.SplitN(strings.Trim(p, "/"), "/", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

// pathHTML highlights parameter segments (:id, *path) inside a route path.
func pathHTML(p string) template.HTML {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") || strings.HasPrefix(s, "*") {
			segs[i] = `<span class="p">` + template.HTMLEscapeString(s) + `</span>`
		}
	}
	return template.HTML(strings.Join(segs, "/"))
}

// renderHTML executes a docs page template into the response.
func renderHTML(c *gin.Context, status int, tmpl *template.Template, data any) {
	// gin presets 404 for NoRoute handlers; make the intended status explicit.
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		// Headers may already be written; log and fall back to a plain error.
		logger.Error("template execute failed", zap.Error(err))
	}
}

// ---- static asset serving (/assets/*) ----

// The ETag is a strong hash over the whole embedded asset tree, computed once
// on first use. Assets are served with `max-age=0, must-revalidate`, so
// browsers revalidate cheaply (304) and can never show stale CSS/JS that came
// from a previous binary.
var (
	assetTagOnce sync.Once
	assetTag     string
)

const docsAssetsPrefix = "/assets/"

func computeAssetETag() string {
	h := sha256.New()
	err := fs.WalkDir(assets, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(assets, p)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", p, len(data))
		h.Write(data)
		return nil
	})
	if err != nil { // cannot happen for embed.FS trees
		logger.Error("hash embedded assets failed", zap.Error(err))
	}
	return `"` + hex.EncodeToString(h.Sum(nil))[:32] + `"`
}

// ServeAssetHandler serves embedded CSS/JS assets with content-addressed ETag
// caching. Registered for GET /assets/*filepath; DocsHandler also delegates
// here so non-GET methods on asset paths stay consistent.
func ServeAssetHandler(c *gin.Context) {
	assetTagOnce.Do(func() { assetTag = computeAssetETag() })

	fp := strings.TrimPrefix(c.Request.URL.Path, docsAssetsPrefix)
	fp = path.Clean(fp)
	if !fs.ValidPath(fp) {
		c.String(http.StatusNotFound, "404 page not found")
		return
	}

	data, err := fs.ReadFile(assets, path.Join("assets", fp))
	if err != nil {
		c.String(http.StatusNotFound, "404 page not found")
		return
	}

	c.Header("Cache-Control", "public, max-age=0, must-revalidate")
	c.Header("ETag", assetTag)
	if c.Request.Header.Get("If-None-Match") == assetTag {
		c.Status(http.StatusNotModified)
		return
	}

	ext := strings.ToLower(path.Ext(fp))
	switch ext {
	case ".css":
		c.Header("Content-Type", "text/css; charset=utf-8")
	case ".js", ".mjs":
		c.Header("Content-Type", "text/javascript; charset=utf-8")
	default:
		if ct := mime.TypeByExtension(ext); ct != "" {
			c.Header("Content-Type", ct)
		}
	}
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(data)
}
