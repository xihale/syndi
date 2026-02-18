package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("expected non-nil client")
	}

	if c.userAgent != "RSSHub-Go/1.0 (+https://github.com/xihale/rsshub-go)" {
		t.Errorf("expected default user agent, got %s", c.userAgent)
	}

	if c.timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", c.timeout)
	}
}

func TestNew_WithOptions(t *testing.T) {
	customUA := "CustomAgent/1.0"
	customTimeout := 60 * time.Second

	c := New(
		WithUserAgent(customUA),
		WithTimeout(customTimeout),
	)

	if c.userAgent != customUA {
		t.Errorf("expected user agent %s, got %s", customUA, c.userAgent)
	}

	if c.timeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, c.timeout)
	}
}

func TestClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		ua := r.Header.Get("User-Agent")
		if !strings.Contains(ua, "RSSHub-Go") {
			t.Errorf("expected User-Agent to contain RSSHub-Go, got %s", ua)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()
	body, err := c.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != "response body" {
		t.Errorf("expected 'response body', got %s", string(body))
	}
}

func TestClient_Get_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()
	_, err := c.Get(ctx, server.URL)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestClient_Get_WithContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := c.Get(ctx, server.URL)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", ct)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("posted"))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()
	body := strings.NewReader("key=value")
	data, err := c.Post(ctx, server.URL, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != "posted" {
		t.Errorf("expected 'posted', got %s", string(data))
	}
}

func TestClient_GetWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customHeader := r.Header.Get("X-Custom")
		if customHeader != "test-value" {
			t.Errorf("expected X-Custom header 'test-value', got %s", customHeader)
		}

		accept := r.Header.Get("Accept")
		if accept != "*/*" {
			t.Errorf("expected Accept '*/*', got %s", accept)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()
	headers := map[string]string{
		"X-Custom": "test-value",
	}

	body, err := c.GetWithHeaders(ctx, server.URL, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != "ok" {
		t.Errorf("expected 'ok', got %s", string(body))
	}
}

func TestClient_SetCookie(t *testing.T) {
	c := New()
	c.SetCookie("example.com", "session", "abc123")

	if c.cookieJar == nil {
		t.Fatal("expected cookie jar to be initialized")
	}

	// Verify cookie was set by checking the jar directly
	u, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cookies := c.cookieJar.Cookies(u.URL)
	if len(cookies) == 0 {
		t.Error("expected cookie to be set")
	}

	if len(cookies) > 0 && cookies[0].Value != "abc123" {
		t.Errorf("expected cookie value 'abc123', got %s", cookies[0].Value)
	}
}

func TestClient_ClearCookies(t *testing.T) {
	c := New()
	c.SetCookie("example.com", "session", "abc123")
	c.ClearCookies()

	// After clearing, cookie should not be sent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err == nil && cookie != nil {
			t.Error("expected no cookie after clearing")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := c.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_WithProxy(t *testing.T) {
	// This test just verifies the option doesn't panic
	c := New(
		WithProxy("http://proxy.example.com:8080"),
	)

	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestClient_FollowRedirects(t *testing.T) {
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("final destination"))
	}))
	defer redirectServer.Close()

	var redirectCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount == 1 {
			http.Redirect(w, r, redirectServer.URL, http.StatusFound)
		}
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()
	body, err := c.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != "final destination" {
		t.Errorf("expected 'final destination', got %s", string(body))
	}
}

func TestClient_Post_BodyReading(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		if string(body) != "test data" {
			t.Errorf("expected 'test data', got %s", string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()
	body := strings.NewReader("test data")
	_, err := c.Post(ctx, server.URL, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_GetWithHeaders_CustomUserAgent(t *testing.T) {
	customUA := "MyBot/1.0"
	c := New(WithUserAgent(customUA))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != customUA {
			t.Errorf("expected User-Agent %s, got %s", customUA, ua)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := c.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Tests for Response Parsing Helpers
// ============================================================================

func TestClient_GetJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "hello", "count": 42}`))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	var result struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	err := c.GetJSON(ctx, server.URL, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Message != "hello" {
		t.Errorf("expected message 'hello', got '%s'", result.Message)
	}

	if result.Count != 42 {
		t.Errorf("expected count 42, got %d", result.Count)
	}
}

func TestClient_GetJSON_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	var result map[string]interface{}
	err := c.GetJSON(ctx, server.URL, &result)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestClient_GetXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><root><message>hello</message><count>42</count></root>`))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	type Result struct {
		Message string `xml:"message"`
		Count   int    `xml:"count"`
	}

	var result Result
	err := c.GetXML(ctx, server.URL, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Message != "hello" {
		t.Errorf("expected message 'hello', got '%s'", result.Message)
	}

	if result.Count != 42 {
		t.Errorf("expected count 42, got %d", result.Count)
	}
}

func TestClient_GetHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Hello World</h1></body></html>`))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	doc, err := c.GetHTML(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	title := doc.Text("h1")
	if title != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", title)
	}
}

func TestClient_GetJSONWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check custom header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Authorization header, got '%s'", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	var result map[string]bool
	headers := map[string]string{
		"Authorization": "Bearer test-token",
	}

	err := c.GetJSONWithHeaders(ctx, server.URL, headers, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result["success"] {
		t.Error("expected success to be true")
	}
}

func TestClient_GetResponseInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", "1234")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	info, err := c.GetResponseInfo(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", info.StatusCode)
	}

	if info.ContentType != "text/html" {
		t.Errorf("expected content type 'text/html', got '%s'", info.ContentType)
	}

	if info.ContentLength != 1234 {
		t.Errorf("expected content length 1234, got %d", info.ContentLength)
	}

	if !info.OK {
		t.Error("expected OK to be true")
	}
}

func TestClient_IsValidURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	// Valid URL
	if !c.IsValidURL(ctx, server.URL) {
		t.Error("expected URL to be valid")
	}

	// Test 404 (still valid URL, just not found)
	server404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server404.Close()

	// 404 is not considered valid (2xx-3xx only)
	if c.IsValidURL(ctx, server404.URL) {
		t.Error("expected 404 URL to be invalid")
	}
}

// ============================================================================
// Tests for Request Builder
// ============================================================================

func TestRequestBuilder_BasicGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	body, err := c.NewRequest("GET", server.URL).
		WithContext(ctx).
		Do()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != `{"result": "success"}` {
		t.Errorf("unexpected response: %s", string(body))
	}
}

func TestRequestBuilder_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		custom := r.Header.Get("X-Custom-Header")
		if custom != "test-value" {
			t.Errorf("expected X-Custom-Header 'test-value', got '%s'", custom)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	_, err := c.NewRequest("GET", server.URL).
		WithContext(ctx).
		WithHeader("X-Custom-Header", "test-value").
		Do()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestBuilder_WithQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("key") != "value" {
			t.Errorf("expected query parameter 'key=value', got '%s'", query.Get("key"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	_, err := c.NewRequest("GET", server.URL).
		WithContext(ctx).
		WithQuery("key", "value").
		Do()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestBuilder_WithBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth to be set")
		}

		if username != "user" || password != "pass" {
			t.Errorf("expected username 'user' and password 'pass', got '%s' and '%s'", username, password)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	_, err := c.NewRequest("GET", server.URL).
		WithContext(ctx).
		WithBasicAuth("user", "pass").
		Do()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestBuilder_DoJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"value": 123}`))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	var result struct {
		Value int `json:"value"`
	}

	err := c.NewRequest("GET", server.URL).
		WithContext(ctx).
		DoJSON(&result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Value != 123 {
		t.Errorf("expected value 123, got %d", result.Value)
	}
}

func TestRequestBuilder_WithJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"received": true}`))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	data := map[string]string{"message": "hello"}

	body, err := c.NewRequest("POST", server.URL).
		WithContext(ctx).
		WithJSON(data)

	if err != nil {
		t.Fatalf("unexpected error from WithJSON: %v", err)
	}

	_, err = body.Do()
	if err != nil {
		t.Fatalf("unexpected error from Do: %v", err)
	}
}

func TestRequestBuilder_DoHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="content">Test Content</div>`))
	}))
	defer server.Close()

	c := New()
	ctx := context.Background()

	doc, err := c.NewRequest("GET", server.URL).
		WithContext(ctx).
		DoHTML()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	content := doc.Text(".content")
	if content != "Test Content" {
		t.Errorf("expected 'Test Content', got '%s'", content)
	}
}
