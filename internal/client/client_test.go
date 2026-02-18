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

	if c.userAgent != "RSSHub-Go/1.0 (+https://github.com/rsshub/go)" {
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
