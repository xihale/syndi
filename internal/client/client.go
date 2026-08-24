package client

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"go.uber.org/zap"
)

// Client wraps an HTTP client with additional features
type Client struct {
	client       *http.Client
	userAgent    string
	timeout      time.Duration
	maxRedirects int
	proxy        *url.URL
	noProxy      bool
	cookieJar    http.CookieJar
	logger       *zap.Logger
	maxRetries   int
	baseDelay    time.Duration
	maxDelay     time.Duration
	bearerToken  string

	// Rate limiting per host
	rateLimiters sync.Map // map[string]*rateLimiter
}

// rateLimiter implements simple token bucket rate limiting
type rateLimiter struct {
	rate       float64 // tokens per second
	capacity   int     // max tokens
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

func newRateLimiter(rate float64, capacity int) *rateLimiter {
	return &rateLimiter{
		rate:       rate,
		capacity:   capacity,
		tokens:     float64(capacity),
		lastRefill: time.Now(),
	}
}

func (rl *rateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(rl.lastRefill).Seconds()
		rl.tokens = rl.tokens + elapsed*rl.rate
		if rl.tokens > float64(rl.capacity) {
			rl.tokens = float64(rl.capacity)
		}
		rl.lastRefill = now

		if rl.tokens >= 1 {
			rl.tokens -= 1
			rl.mu.Unlock()
			return nil
		}
		need := 1 - rl.tokens
		waitDuration := time.Duration(need/rl.rate*float64(time.Second)) + time.Millisecond
		rl.mu.Unlock()

		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// getRateLimiter returns or creates a rate limiter for the given host
func (c *Client) getRateLimiter(host string) *rateLimiter {
	if limiter, ok := c.rateLimiters.Load(host); ok {
		return limiter.(*rateLimiter)
	}
	// Default: 10 requests per second with burst of 10
	limiter := newRateLimiter(10, 10)
	c.rateLimiters.Store(host, limiter)
	return limiter
}

// ClientOption configures the client
type ClientOption func(*Client)

// New creates a new HTTP client
func New(options ...ClientOption) *Client {
	c := &Client{
		userAgent:    "Syndi/0.0.1 (+https://github.com/xihale/syndi)",
		timeout:      30 * time.Second,
		maxRedirects: 10,
		maxRetries:   3,
		baseDelay:    1 * time.Second,
		maxDelay:     30 * time.Second,
	}

	for _, opt := range options {
		opt(c)
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if c.noProxy {
		transport.Proxy = nil
	} else if c.proxy != nil {
		transport.Proxy = http.ProxyURL(c.proxy)
	}

	if c.cookieJar == nil {
		jar, _ := cookiejar.New(nil)
		c.cookieJar = jar
	}

	c.client = &http.Client{
		Transport:     transport,
		Timeout:       c.timeout,
		CheckRedirect: c.checkRedirect,
		Jar:           c.cookieJar,
	}

	return c
}

// WithUserAgent sets the user agent
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithProxy sets a proxy URL
func WithProxy(proxyURL string) ClientOption {
	return func(c *Client) {
		if u, err := url.Parse(proxyURL); err == nil {
			c.proxy = u
		}
	}
}

// WithNoProxy forces the client to skip proxy usage (including env proxies)
func WithNoProxy() ClientOption {
	return func(c *Client) {
		c.proxy = nil
		c.noProxy = true
	}
}

// WithCookieJar sets the cookie jar
func WithCookieJar(jar http.CookieJar) ClientOption {
	return func(c *Client) {
		c.cookieJar = jar
	}
}

// WithLogger sets the logger for request/response logging
func WithLogger(logger *zap.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithBearerToken sets the Authorization header with Bearer token
func WithBearerToken(token string) ClientOption {
	return func(c *Client) {
		c.bearerToken = token
	}
}

// WithRetryConfig configures retry behavior
func WithRetryConfig(maxRetries int, baseDelay, maxDelay time.Duration) ClientOption {
	return func(c *Client) {
		if maxRetries > 0 {
			c.maxRetries = maxRetries
		}
		if baseDelay > 0 {
			c.baseDelay = baseDelay
		}
		if maxDelay > 0 {
			c.maxDelay = maxDelay
		}
	}
}

// WithMaxRedirects sets the maximum number of redirects to follow
func WithMaxRedirects(maxRedirects int) ClientOption {
	return func(c *Client) {
		if maxRedirects > 0 {
			c.maxRedirects = maxRedirects
		}
	}
}

// WithRateLimit sets rate limit (requests per second) for specific host
func WithRateLimit(host string, rps float64, burst int) ClientOption {
	return func(c *Client) {
		if burst <= 0 {
			burst = int(rps)
		}
		limiter := newRateLimiter(rps, burst)
		c.rateLimiters.Store(host, limiter)
	}
}

// doRequestWithRetry performs an HTTP request with exponential backoff retry
func (c *Client) doRequestWithRetry(ctx context.Context, req *http.Request) ([]byte, error) {
	// Set default headers
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
	if c.bearerToken != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}

	var lastErr error
	delay := c.baseDelay
	maxAttempts := 1
	if isRetryableMethod(req.Method) && isRequestReplayable(req) {
		maxAttempts = c.maxRetries + 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to reset request body: %w", err)
			}
			req.Body = body
		}

		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			if c.logger != nil {
				c.logger.Warn("Retrying request",
					zap.String("url", req.URL.String()),
					zap.Int("attempt", attempt),
					zap.Duration("delay", delay))
			}
		}

		// Apply rate limiting per host
		host := req.URL.Host
		limiter := c.getRateLimiter(host)
		if err := limiter.Wait(ctx); err != nil {
			return nil, err
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to fetch %s: %w", req.URL.String(), err)
			delay = minDuration(delay*2, c.maxDelay)
			continue
		}

		// Check for rate limiting
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			retryAfter := resp.Header.Get("Retry-After")
			if retryAfter != "" {
				if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil && seconds > 0 {
					delay = time.Duration(seconds) * time.Second
				}
			}
			lastErr = fmt.Errorf("failed to fetch %s: status %s (rate limited)", req.URL.String(), resp.Status)
			_ = resp.Body.Close()
			if shouldRetryStatus(resp.StatusCode) {
				delay = minDuration(delay*2, c.maxDelay)
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			if readErr == nil && len(bodyBytes) > 0 {
				lastErr = fmt.Errorf("failed to fetch %s: status %s, response: %s", req.URL.String(), resp.Status, string(bodyBytes))
			} else {
				lastErr = fmt.Errorf("failed to fetch %s: status %s", req.URL.String(), resp.Status)
			}
			_ = resp.Body.Close()
			if shouldRetryStatus(resp.StatusCode) {
				delay = minDuration(delay*2, c.maxDelay)
				continue
			}
			return nil, lastErr
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			delay = minDuration(delay*2, c.maxDelay)
			continue
		}

		if c.logger != nil {
			c.logger.Debug("HTTP request successful",
				zap.String("url", req.URL.String()),
				zap.Int("status", resp.StatusCode),
				zap.Int("bytes", len(body)))
		}

		return body, nil
	}

	return nil, lastErr
}

// Get performs a GET request with retry logic
func (c *Client) Get(ctx context.Context, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	return c.doRequestWithRetry(ctx, req)
}

// GetWithQuery performs a GET request with query parameters
func (c *Client) GetWithQuery(ctx context.Context, urlStr string, queryParams map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	if len(queryParams) > 0 {
		q := req.URL.Query()
		for k, v := range queryParams {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	return c.doRequestWithRetry(ctx, req)
}

// Post performs a POST request with retry logic
func (c *Client) Post(ctx context.Context, urlStr string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.doRequestWithRetry(ctx, req)
}

// PostWithHeaders performs a POST request with a full body and explicit headers.
func (c *Client) PostWithHeaders(ctx context.Context, urlStr string, body []byte, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return c.doRequestWithRetry(ctx, req)
}

// GetWithHeaders performs a GET request with custom headers
func (c *Client) GetWithHeaders(ctx context.Context, urlStr string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.doRequestWithRetry(ctx, req)
}

func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= c.maxRedirects {
		return errors.New("stopped after too many redirects")
	}

	// Copy User-Agent from original request
	if len(via) > 0 {
		req.Header.Set("User-Agent", c.userAgent)
	}
	return nil
}

func isRequestReplayable(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return req.GetBody != nil
}

func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

func shouldRetryStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, // 408
		http.StatusTooEarly,            // 425
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// SetCookie adds a cookie for a domain
func (c *Client) SetCookie(domain, name, value string) {
	if c.cookieJar != nil {
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}
		u, _ := url.Parse(domain)
		c.cookieJar.SetCookies(u, []*http.Cookie{
			{
				Name:   name,
				Value:  value,
				Domain: u.Host,
				Path:   "/",
			},
		})
	}
}

// ClearCookies clears all cookies
func (c *Client) ClearCookies() {
	if c.cookieJar != nil {
		c.cookieJar = nil
		jar, _ := cookiejar.New(nil)
		c.cookieJar = jar
	}
}

// minDuration returns the minimum of two durations
func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// Response Parsing Helpers
// ============================================================================

// GetJSON performs a GET request and parses JSON response into target
func (c *Client) GetJSON(ctx context.Context, urlStr string, target interface{}) error {
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// GetJSONWithHeaders performs a GET request with custom headers and parses JSON
func (c *Client) GetJSONWithHeaders(ctx context.Context, urlStr string, headers map[string]string, target interface{}) error {
	body, err := c.GetWithHeaders(ctx, urlStr, headers)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// GetXML performs a GET request and parses XML response into target
func (c *Client) GetXML(ctx context.Context, urlStr string, target interface{}) error {
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return err
	}
	return xml.Unmarshal(body, target)
}

// GetXMLWithHeaders performs a GET request with custom headers and parses XML
func (c *Client) GetXMLWithHeaders(ctx context.Context, urlStr string, headers map[string]string, target interface{}) error {
	body, err := c.GetWithHeaders(ctx, urlStr, headers)
	if err != nil {
		return err
	}
	return xml.Unmarshal(body, target)
}

// GetHTML performs a GET request and returns parsed HTML document
func (c *Client) GetHTML(ctx context.Context, urlStr string) (*parser.Document, error) {
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, err
	}
	return parser.LoadString(string(body))
}

// GetHTMLWithHeaders performs a GET request with custom headers and returns parsed HTML
func (c *Client) GetHTMLWithHeaders(ctx context.Context, urlStr string, headers map[string]string) (*parser.Document, error) {
	body, err := c.GetWithHeaders(ctx, urlStr, headers)
	if err != nil {
		return nil, err
	}
	return parser.LoadString(string(body))
}

// ============================================================================
// Response Validation Helpers
// ============================================================================

// ResponseInfo contains metadata about an HTTP response
type ResponseInfo struct {
	StatusCode    int
	Header        http.Header
	ContentType   string
	ContentLength int64
	OK            bool
}

// GetResponseInfo performs a HEAD request to get response metadata
func (c *Client) GetResponseInfo(ctx context.Context, urlStr string) (*ResponseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, urlStr, nil)
	if err != nil {
		return nil, err
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	info := &ResponseInfo{
		StatusCode:    resp.StatusCode,
		Header:        resp.Header,
		ContentLength: resp.ContentLength,
		OK:            resp.StatusCode == http.StatusOK,
	}

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		// Parse content type (e.g., "text/html; charset=utf-8")
		if parts := strings.SplitN(ct, ";", 2); len(parts) > 0 {
			info.ContentType = strings.TrimSpace(parts[0])
		}
	}

	return info, nil
}

// IsValidURL checks if a URL is accessible and returns appropriate status
func (c *Client) IsValidURL(ctx context.Context, urlStr string) bool {
	info, err := c.GetResponseInfo(ctx, urlStr)
	if err != nil {
		return false
	}
	return info.StatusCode >= 200 && info.StatusCode < 400
}

// ============================================================================
// Request Builder (Fluent API)
// ============================================================================

// RequestBuilder provides a fluent API for building HTTP requests
type RequestBuilder struct {
	client    *Client
	ctx       context.Context
	method    string
	url       string
	headers   map[string]string
	query     map[string]string
	body      io.Reader
	basicAuth struct{ username, password string }
}

// NewRequest creates a new request builder
func (c *Client) NewRequest(method, urlStr string) *RequestBuilder {
	return &RequestBuilder{
		client:  c,
		ctx:     context.Background(),
		method:  method,
		url:     urlStr,
		headers: make(map[string]string),
		query:   make(map[string]string),
	}
}

// WithContext sets the request context
func (rb *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	rb.ctx = ctx
	return rb
}

// WithHeader adds a header to the request
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

// WithHeaders adds multiple headers
func (rb *RequestBuilder) WithHeaders(headers map[string]string) *RequestBuilder {
	for k, v := range headers {
		rb.headers[k] = v
	}
	return rb
}

// WithQuery adds a query parameter
func (rb *RequestBuilder) WithQuery(key, value string) *RequestBuilder {
	rb.query[key] = value
	return rb
}

// WithQueryMap adds multiple query parameters
func (rb *RequestBuilder) WithQueryMap(params map[string]string) *RequestBuilder {
	for k, v := range params {
		rb.query[k] = v
	}
	return rb
}

// WithBody sets the request body
func (rb *RequestBuilder) WithBody(body io.Reader) *RequestBuilder {
	rb.body = body
	return rb
}

// WithJSON sets the request body as JSON
func (rb *RequestBuilder) WithJSON(data interface{}) (*RequestBuilder, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	rb.body = strings.NewReader(string(jsonData))
	rb.headers["Content-Type"] = "application/json"
	return rb, nil
}

// WithBasicAuth sets basic authentication
func (rb *RequestBuilder) WithBasicAuth(username, password string) *RequestBuilder {
	rb.basicAuth.username = username
	rb.basicAuth.password = password
	return rb
}

// Do executes the request and returns raw response
func (rb *RequestBuilder) Do() ([]byte, error) {
	req, err := http.NewRequestWithContext(rb.ctx, rb.method, rb.url, rb.body)
	if err != nil {
		return nil, err
	}

	// Apply headers
	for k, v := range rb.headers {
		req.Header.Set(k, v)
	}

	// Apply query parameters
	if len(rb.query) > 0 {
		q := req.URL.Query()
		for k, v := range rb.query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	// Apply basic auth
	if rb.basicAuth.username != "" {
		req.SetBasicAuth(rb.basicAuth.username, rb.basicAuth.password)
	}

	return rb.client.doRequestWithRetry(rb.ctx, req)
}

// DoJSON executes the request and parses JSON response
func (rb *RequestBuilder) DoJSON(target interface{}) error {
	body, err := rb.Do()
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// DoXML executes the request and parses XML response
func (rb *RequestBuilder) DoXML(target interface{}) error {
	body, err := rb.Do()
	if err != nil {
		return err
	}
	return xml.Unmarshal(body, target)
}

// DoHTML executes the request and returns parsed HTML document
func (rb *RequestBuilder) DoHTML() (*parser.Document, error) {
	body, err := rb.Do()
	if err != nil {
		return nil, err
	}
	return parser.LoadString(string(body))
}
