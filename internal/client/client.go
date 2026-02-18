package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Client wraps an HTTP client with additional features
type Client struct {
	client       *http.Client
	userAgent    string
	timeout      time.Duration
	maxRedirects int
	proxy        *url.URL
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
		userAgent:    "RSSHub-Go/1.0 (+https://github.com/rsshub/go)",
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
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if c.proxy != nil {
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

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
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
		defer resp.Body.Close()

		// Check for rate limiting
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			retryAfter := resp.Header.Get("Retry-After")
			if retryAfter != "" {
				if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil && seconds > 0 {
					delay = time.Duration(seconds) * time.Second
				}
			}
			lastErr = fmt.Errorf("failed to fetch %s: status %s (rate limited)", req.URL.String(), resp.Status)
			delay = minDuration(delay*2, c.maxDelay)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			if readErr == nil && len(bodyBytes) > 0 {
				lastErr = fmt.Errorf("failed to fetch %s: status %s, response: %s", req.URL.String(), resp.Status, string(bodyBytes))
			} else {
				lastErr = fmt.Errorf("failed to fetch %s: status %s", req.URL.String(), resp.Status)
			}
			delay = minDuration(delay*2, c.maxDelay)
			continue
		}

		body, err := io.ReadAll(resp.Body)
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
	// Copy User-Agent from original request
	if len(via) > 0 {
		req.Header.Set("User-Agent", c.userAgent)
	}
	return nil
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
