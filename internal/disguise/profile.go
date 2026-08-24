// Package disguise provides a unified request-disguise API for routes.
//
// Common needs distilled from ported routes:
//   - browser-realistic User-Agent and header set (many sites 403 default UAs)
//   - Referer / Accept-Language tuning for region-locked or hotlink-checked sites
//   - raw Cookie strings for semi-private endpoints
//   - User-Agent rotation to spread out fingerprinting
//   - optional politeness delay between requests
//
// The API is a fluent builder that produces plain HTTP headers, so it composes
// with the shared client (retry, proxy, rate limiting stay intact):
//
//	doc, err := disguise.Chrome().Lang("zh-CN").
//	    Referer("https://example.com/").
//	    GetHTML(ctx, c.Client(), targetURL)
package disguise

import (
	"hash/fnv"
	"math/rand"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// RotateStrategy selects how a profile picks a User-Agent from its pool.
type RotateStrategy int

const (
	// RotateRoundRobin cycles the pool deterministically across all requests.
	RotateRoundRobin RotateStrategy = iota
	// RotateRandom picks uniformly at random per request.
	RotateRandom
	// RotateStickyPerHost pins one agent per target host (consistent fingerprint).
	RotateStickyPerHost
)

// Profile describes how to disguise outgoing requests.
type Profile struct {
	name       string
	userAgents []string
	rotate     RotateStrategy
	headers    map[string]string

	cookie    string
	referer   string
	language  string
	delayMin  time.Duration
	delayMax  time.Duration
	acceptSet bool

	mu      sync.Mutex
	counter atomic.Uint64
}

// newProfile creates a base profile with sane defaults.
func newProfile(name string) *Profile {
	return &Profile{
		name:    name,
		headers: make(map[string]string),
		rotate:  RotateStickyPerHost,
	}
}

// Chrome returns a desktop Chrome profile: realistic UA plus Sec-Fetch /
// sec-ch-ua hints that modern sites expect from real browsers.
func Chrome() *Profile {
	p := newProfile("chrome")
	p.userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	}
	p.headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
	p.headers["Sec-Fetch-Dest"] = "document"
	p.headers["Sec-Fetch-Mode"] = "navigate"
	p.headers["Sec-Fetch-Site"] = "none"
	p.headers["Sec-Fetch-User"] = "?1"
	p.headers["Upgrade-Insecure-Requests"] = "1"
	p.acceptSet = true
	return p
}

// Firefox returns a desktop Firefox profile.
func Firefox() *Profile {
	p := newProfile("firefox")
	p.userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0",
	}
	p.headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
	p.headers["Upgrade-Insecure-Requests"] = "1"
	p.acceptSet = true
	return p
}

// Safari returns a macOS Safari profile.
func Safari() *Profile {
	p := newProfile("safari")
	p.userAgents = []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
	}
	p.headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	p.acceptSet = true
	return p
}

// ChromeMobile returns an Android Chrome (Pixel) mobile profile.
func ChromeMobile() *Profile {
	p := newProfile("chrome-mobile")
	p.userAgents = []string{
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36",
	}
	p.headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
	p.headers["Sec-Fetch-Dest"] = "document"
	p.headers["Sec-Fetch-Mode"] = "navigate"
	p.headers["Sec-Fetch-Site"] = "none"
	p.headers["Upgrade-Insecure-Requests"] = "1"
	p.acceptSet = true
	return p
}

// Custom returns an empty profile carrying no identity of its own except the
// given user agents; combine with WithHeader for full control.
func Custom(userAgents ...string) *Profile {
	p := newProfile("custom")
	if len(userAgents) > 0 {
		p.userAgents = append([]string{}, userAgents...)
	}
	return p
}

// --- Fluent modifiers (all return the same profile for chaining) ---

// Rotate overrides the User-Agent rotation strategy.
func (p *Profile) Rotate(strategy RotateStrategy) *Profile {
	p.rotate = strategy
	return p
}

// WithHeader sets a static header on every request. It overrides preset values;
// later calls with the same key win. Use "" to delete a header from the set.
func (p *Profile) WithHeader(key, value string) *Profile {
	if value == "" {
		delete(p.headers, key)
		if key == "Accept" {
			p.acceptSet = false
		}
		return p
	}
	p.headers[key] = value
	if key == "Accept" {
		p.acceptSet = true
	}
	return p
}

// Accept overrides the Accept header.
func (p *Profile) Accept(value string) *Profile {
	return p.WithHeader("Accept", value)
}

// JSONAccept tunes the profile for JSON API endpoints (XHR-like).
func (p *Profile) JSONAccept() *Profile {
	p.WithHeader("Accept", "application/json, text/plain, */*")
	p.WithHeader("Sec-Fetch-Dest", "empty")
	p.WithHeader("Sec-Fetch-Mode", "cors")
	return p
}

// Referer sets the Referer header (pass full URL or origin).
func (p *Profile) Referer(referer string) *Profile {
	p.referer = referer
	return p
}

// Lang sets Accept-Language (e.g. "zh-CN,zh;q=0.9,en;q=0.8").
func (p *Profile) Lang(language string) *Profile {
	p.language = language
	return p
}

// Cookie sets a raw Cookie header string ("k1=v1; k2=v2"). Takes precedence
// over the client's cookie jar for these requests.
func (p *Profile) Cookie(rawCookie string) *Profile {
	p.cookie = rawCookie
	return p
}

// Delay adds a random politeness sleep in [min,max] before each request.
// Zero durations disable it (the default).
func (p *Profile) Delay(min, max time.Duration) *Profile {
	if min < 0 {
		min = 0
	}
	if max < min {
		max = min
	}
	p.delayMin, p.delayMax = min, max
	return p
}

// --- Header materialization ---

// Headers materializes the final header set for the target URL. Values set by
// dedicated builder methods (Referer/Lang/Cookie) win over static headers;
// explicit WithHeader calls win over everything.
func (p *Profile) Headers(targetURL string) map[string]string {
	out := make(map[string]string, len(p.headers)+4)
	for k, v := range p.headers {
		out[k] = v
	}
	if !p.acceptSet {
		out["Accept"] = "*/*"
	}
	if ua := p.pickUserAgent(targetURL); ua != "" {
		out["User-Agent"] = ua
	}
	if p.referer != "" {
		out["Referer"] = p.referer
	} else if v, ok := out["Referer"]; !ok || v == "" {
		// Sensible same-site referer reduces hotlink rejections when unset.
		if u, err := url.Parse(targetURL); err == nil && u.Scheme != "" && u.Host != "" {
			base := u.Scheme + "://" + u.Host + "/"
			out["Referer"] = base
		}
	}
	if p.language != "" {
		out["Accept-Language"] = p.language
	}
	if p.cookie != "" {
		out["Cookie"] = p.cookie
	}
	return out
}

// Name reports the profile's preset name.
func (p *Profile) Name() string { return p.name }

func (p *Profile) pickUserAgent(targetURL string) string {
	if len(p.userAgents) == 0 {
		return ""
	}
	if len(p.userAgents) == 1 {
		return p.userAgents[0]
	}
	switch p.rotate {
	case RotateRandom:
		return p.userAgents[rand.Intn(len(p.userAgents))]
	case RotateStickyPerHost:
		return p.userAgents[stickyIndex(targetURL, len(p.userAgents))]
	default: // RotateRoundRobin
		n := p.counter.Add(1) - 1
		return p.userAgents[int(n)%len(p.userAgents)]
	}
}

func stickyIndex(targetURL string, n int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(hostOf(targetURL)))
	return int(h.Sum32() % uint32(n))
}

func hostOf(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return u.Host
	}
	return rawURL
}

// nap sleeps for the configured random delay if enabled.
func (p *Profile) nap() {
	if p.delayMax <= 0 {
		return
	}
	span := p.delayMax - p.delayMin
	d := p.delayMin
	if span > 0 {
		d += time.Duration(rand.Int63n(int64(span)))
	}
	if d > 0 {
		time.Sleep(d)
	}
}
