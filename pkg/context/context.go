package contextpkg

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/pkg/cache"
)

// Context wraps the HTTP request with utilities for route handlers
type Context struct {
	Req    *http.Request
	Writer http.ResponseWriter
	Params map[string]string
	Query  url.Values
	cache  cache.Cache
	client *client.Client
}

// NewContext creates a new context from an HTTP request
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	params := make(map[string]string)

	// Extract params if stored in context by Gin
	if ctxParams, ok := r.Context().Value("params").(map[string]string); ok {
		params = ctxParams
	}

	return &Context{
		Req:    r,
		Writer: w,
		Params: params,
		Query:  r.URL.Query(),
	}
}

// SetParams sets the path parameters
func (c *Context) SetParams(params map[string]string) {
	c.Params = params
}

// Param retrieves a path parameter
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// QueryParam retrieves a query parameter
func (c *Context) QueryParam(key string) string {
	return c.Query.Get(key)
}

// Client returns the HTTP client for making requests
func (c *Context) Client() *client.Client {
	return c.client
}

// SetClient sets the HTTP client
func (c *Context) SetClient(client *client.Client) {
	c.client = client
}

// Cache returns the cache instance
func (c *Context) Cache() cache.Cache {
	return c.cache
}

// SetCache sets the cache instance
func (c *Context) SetCache(cache cache.Cache) {
	c.cache = cache
}

// CacheTryGet attempts to retrieve from cache, or computes and stores if missing
func (c *Context) CacheTryGet(key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if val, ok := c.cache.Get(key); ok {
		return val, nil
	}

	val, err := fn()
	if err != nil {
		return nil, err
	}

	c.cache.Set(key, val, ttl)
	return val, nil
}

// BaseURL returns the base URL of the request
func (c *Context) BaseURL() string {
	scheme := "http"
	if c.Req.TLS != nil || c.Req.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := c.Req.Host
	return scheme + "://" + host
}

// GetHeader retrieves a request header
func (c *Context) GetHeader(key string) string {
	return c.Req.Header.Get(key)
}

// SetHeader sets a response header
func (c *Context) SetHeader(key, value string) {
	c.Writer.Header().Set(key, value)
}

// Parent returns the parent context
func (c *Context) Parent() context.Context {
	return c.Req.Context()
}
