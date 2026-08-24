package disguise

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xihale/rsshub-go/internal/client"
)

func TestChromePresetHeaders(t *testing.T) {
	p := Chrome()
	h := p.Headers("https://example.org/page")
	if ua, ok := h["User-Agent"]; !ok || ua == "" {
		t.Fatal("chrome preset must set User-Agent")
	}
	for _, key := range []string{"Accept", "Sec-Fetch-Dest", "Upgrade-Insecure-Requests"} {
		if h[key] == "" {
			t.Fatalf("chrome preset missing %s", key)
		}
	}
	// Same-site referer default
	if want := "https://example.org/"; h["Referer"] != want {
		t.Fatalf("referer default = %q want %q", h["Referer"], want)
	}
}

func TestBuilderOverrides(t *testing.T) {
	p := Firefox().Lang("zh-CN,zh;q=0.9").Cookie("sid=abc").
		Referer("https://ref.example/").WithHeader("X-Test", "1")
	h := p.Headers("https://target.example/api")
	if h["Accept-Language"] != "zh-CN,zh;q=0.9" {
		t.Fatalf("lang: %v", h)
	}
	if h["Cookie"] != "sid=abc" {
		t.Fatalf("cookie: %v", h)
	}
	if h["Referer"] != "https://ref.example/" {
		t.Fatalf("referer override: %v", h)
	}
	if h["X-Test"] != "1" {
		t.Fatalf("custom header: %v", h)
	}
	// WithHeader wins over dedicated setters
	p.Referer("https://final.example/")
	if p.Headers("https://x.test/")["Referer"] != "https://final.example/" {
		t.Fatal("WithHeader should beat dedicated setter")
	}
}

func TestRotationStrategies(t *testing.T) {
	p := Custom("ua-1", "ua-2", "ua-3")

	// Round robin cycles deterministically
	p.Rotate(RotateRoundRobin)
	seen := map[string]bool{}
	for i := 0; i < 6; i++ {
		seen[p.Headers("https://rr.test/")["User-Agent"]] = true
	}
	if len(seen) != 3 {
		t.Fatalf("round robin should cycle all agents, saw %v", seen)
	}

	// Sticky per host is stable per host and varies across hosts
	p.Rotate(RotateStickyPerHost)
	first := p.Headers("https://sticky.test/a")["User-Agent"]
	for i := 0; i < 5; i++ {
		if got := p.Headers("https://sticky.test/b")["User-Agent"]; got != first {
			t.Fatalf("sticky per host changed: %s vs %s", got, first)
		}
	}

	// Random eventually varies (statistically certain with 3 options x 30 draws)
	p.Rotate(RotateRandom)
	distinct := map[string]bool{}
	for i := 0; i < 30; i++ {
		distinct[p.Headers("https://rand.test/")["User-Agent"]] = true
	}
	if len(distinct) < 2 {
		t.Fatal("random rotation never varied")
	}
}

func TestJSONAcceptAndDelete(t *testing.T) {
	p := Chrome().JSONAccept()
	h := p.Headers("https://api.test/x")
	if h["Accept"] != "application/json, text/plain, */*" {
		t.Fatalf("json accept: %v", h["Accept"])
	}
	if h["Sec-Fetch-Dest"] != "empty" {
		t.Fatalf("xhr dest: %v", h["Sec-Fetch-Dest"])
	}
	p.WithHeader("Accept", "")
	if p.Headers("https://api.test/x")["Accept"] != "*/*" {
		t.Fatal("deleted Accept should fall back to */*")
	}
}

func TestRequestAgainstServer(t *testing.T) {
	var gotUA, gotLang, gotCookie, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotLang = r.Header.Get("Accept-Language")
		gotCookie = r.Header.Get("Cookie")
		gotMethod = r.Method
		if r.Method == http.MethodPost {
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			gotBody = string(buf)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "yes"})
	}))
	defer srv.Close()

	cl := client.New(client.WithTimeout(10 * time.Second))
	ctx := context.Background()

	var resp struct {
		OK string `json:"ok"`
	}
	p := Chrome().Lang("en-US").Cookie("a=b")
	if err := p.Fetch(srv.URL).GetJSON(ctx, cl, &resp); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if resp.OK != "yes" {
		t.Fatalf("bad body: %v", resp)
	}
	if gotUA == "" || gotUA == "RSSHub-Go/1.0 (+https://github.com/xihale/rsshub-go)" {
		t.Fatalf("disguised UA not applied: %q", gotUA)
	}
	if gotLang != "en-US" || gotCookie != "a=b" {
		t.Fatalf("headers not applied: lang=%q cookie=%q", gotLang, gotCookie)
	}

	// POST JSON path
	err := p.PostJSON(srv.URL+"/post", map[string]int{"n": 1}).GetJSON(ctx, cl, &resp)
	if err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotBody == "" {
		t.Fatal("POST body empty")
	}
}

func TestDelayBounds(t *testing.T) {
	p := Chrome().Delay(0, time.Millisecond)
	start := time.Now()
	p.nap() // should return within ~ms
	if time.Since(start) > 200*time.Millisecond {
		t.Fatal("delay too long")
	}
	p.Delay(50*time.Millisecond, -1) // max<min clamps to min
	if p.delayMin != 50*time.Millisecond || p.delayMax != 50*time.Millisecond {
		t.Fatalf("clamp broken: %v %v", p.delayMin, p.delayMax)
	}
}
