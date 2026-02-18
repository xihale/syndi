package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Client wraps an HTTP client with additional features
type Client struct {
	client     *http.Client
	userAgent   string
	timeout     time.Duration
	maxRedirects int
	proxy       *url.URL
	cookieJar   http.CookieJar
}

// ClientOption configures the client
type ClientOption func(*Client)

// New creates a new HTTP client
func New(options ...ClientOption) *Client {
	c := &Client{
		userAgent:   "RSSHub-Go/1.0 (+https://github.com/rsshub/go)",
		timeout:     30 * time.Second,
		maxRedirects: 10,
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

// Get performs a GET request
func (c *Client) Get(ctx context.Context, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: status %s", urlStr, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, urlStr string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to post to %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to post to %s: status %s", urlStr, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// GetWithHeaders performs a GET request with custom headers
func (c *Client) GetWithHeaders(ctx context.Context, urlStr string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "*/*")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: status %s", urlStr, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	// Copy cookies from previous responses
	if len(via) > 0 {
		req.Header.Set("User-Agent", c.userAgent)
	}
	return nil
}

// SetCookie adds a cookie for a domain
func (c *Client) SetCookie(domain, name, value string) {
	if c.cookieJar != nil {
		// Ensure domain has scheme
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
